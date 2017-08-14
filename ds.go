package kail

import (
	"context"

	logutil "github.com/boz/go-logutil"
	"github.com/boz/kcache/filter"
	"github.com/boz/kcache/nsname"
	"github.com/boz/kcache/types/node"
	"github.com/boz/kcache/types/pod"
	"github.com/boz/kcache/types/service"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type DSBuilder interface {
	WithNamespace(name ...string) DSBuilder
	WithPods(id ...nsname.NSName) DSBuilder
	WithLabels(labels map[string]string) DSBuilder
	WithService(id ...nsname.NSName) DSBuilder
	WithNode(name ...string) DSBuilder

	Create(ctx context.Context, cs kubernetes.Interface) (DS, error)
}

type DS interface {
	Ready() <-chan struct{}
	Stop()
}

type dsBuilder struct {
	namespaces []string
	pods       []nsname.NSName
	labels     map[string]string
	services   []nsname.NSName
	nodes      []string
}

func NewDSBuilder() DSBuilder {
	return &dsBuilder{labels: make(map[string]string)}
}

func (b *dsBuilder) WithNamespace(name ...string) DSBuilder {
	b.namespaces = append(b.namespaces, name...)
	return b
}

func (b *dsBuilder) WithPods(id ...nsname.NSName) DSBuilder {
	b.pods = append(b.pods, id...)
	return b
}

func (b *dsBuilder) WithLabels(labels map[string]string) DSBuilder {
	for k, v := range labels {
		b.labels[k] = v
	}
	return b
}

func (b *dsBuilder) WithService(id ...nsname.NSName) DSBuilder {
	b.services = append(b.services, id...)
	return b
}

func (b *dsBuilder) WithNode(name ...string) DSBuilder {
	b.nodes = append(b.nodes, name...)
	return b
}

func (b *dsBuilder) Create(ctx context.Context, cs kubernetes.Interface) (DS, error) {
	log := logutil.FromContextOrDefault(ctx)

	ds := &datastore{}

	base, err := pod.NewController(ctx, log, cs, "")
	if err != nil {
		return nil, log.Err(err, "base pod controller")
	}

	ds.podBase = base
	ds.pods, err = base.CloneWithFilter(filter.Null())
	if err != nil {
		ds.closeAll()
		return nil, log.Err(err, "null filter")
	}

	if sz := len(b.namespaces); sz > 0 {
		ids := make([]nsname.NSName, 0, sz)
		for _, ns := range b.namespaces {
			ids = append(ids, nsname.New(ns, ""))
		}

		ds.pods, err = ds.pods.CloneWithFilter(filter.NSName(ids...))
		if err != nil {
			ds.closeAll()
			return nil, log.Err(err, "namespace filter")
		}
	}

	if len(b.pods) != 0 {
		ds.pods, err = ds.pods.CloneWithFilter(filter.NSName(b.pods...))
		if err != nil {
			ds.closeAll()
			return nil, log.Err(err, "pods filter")
		}
	}

	if len(b.labels) != 0 {
		ds.pods, err = ds.pods.CloneWithFilter(filter.Labels(b.labels))
		if err != nil {
			ds.closeAll()
			return nil, log.Err(err, "labels filter")
		}
	}

	if len(b.services) != 0 {
		ds.servicesBase, err = service.NewController(ctx, log, cs, "")
		if err != nil {
			ds.closeAll()
			return nil, log.Err(err, "service base controller")
		}

		ds.services, err = ds.servicesBase.CloneWithFilter(filter.NSName(b.services...))
		if err != nil {
			ds.closeAll()
			return nil, log.Err(err, "service controller")
		}

		pods, err := ds.pods.CloneWithFilter(filter.All())
		if err != nil {
			ds.closeAll()
			return nil, log.Err(err, "services filter")
		}

		ds.pods = pods

		update := func(_ *v1.Service) {
			objs, err := ds.services.Cache().List()
			if err == nil {
				log.Err(err, "service cache list")
				return
			}
			pods.Refilter(service.PodsFilter(objs...))
		}

		handler := service.BuildHandler().
			OnInitialize(func(objs []*v1.Service) {
				pods.Refilter(service.PodsFilter(objs...))
			}).
			OnCreate(update).
			OnUpdate(update).
			OnDelete(update).
			Create()

		if _, err := service.NewMonitor(ds.services, handler); err != nil {
			ds.closeAll()
			return nil, log.Err(err, "services monitor")
		}
	}

	return ds, nil
}

type datastore struct {
	podBase      pod.Controller
	servicesBase service.Controller
	nodesBase    node.Controller
	pods         pod.FilterController
	services     service.Controller
	nodes        node.Controller
}

func (ds *datastore) Ready() <-chan struct{} {
	return ds.pods.Ready()
}

func (ds *datastore) Stop() {
	ds.closeAll()
	ds.waitAll()
}

func (ds *datastore) closeAll() {
	closeController(ds.podBase)
	closeController(ds.servicesBase)
	closeController(ds.nodesBase)
	closeController(ds.pods)
	closeController(ds.services)
	closeController(ds.nodes)
}

func (ds *datastore) waitAll() {
	waitController(ds.podBase)
	waitController(ds.servicesBase)
	waitController(ds.nodesBase)
	waitController(ds.pods)
	waitController(ds.services)
	waitController(ds.nodes)
}

type closeable interface {
	Close()
}

func closeController(controller closeable) {
	if controller != nil {
		controller.Close()
	}
}

type doneable interface {
	Done() <-chan struct{}
}

func waitController(controller doneable) {
	if controller != nil {
		<-controller.Done()
	}
}
