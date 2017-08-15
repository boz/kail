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
	Pods() pod.Controller
	Ready() <-chan struct{}
	Done() <-chan struct{}
	Shutdown()
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

	ds := &datastore{
		readych: make(chan struct{}),
		donech:  make(chan struct{}),
	}

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

	go ds.waitReadyAll()
	go ds.waitDoneAll()

	return ds, nil
}

type datastore struct {
	podBase      pod.Controller
	servicesBase service.Controller
	nodesBase    node.Controller

	pods     pod.Controller
	services service.Controller
	nodes    node.Controller

	readych chan struct{}
	donech  chan struct{}
}

func (ds *datastore) Pods() pod.Controller {
	return ds.pods
}

func (ds *datastore) Ready() <-chan struct{} {
	return ds.readych
}

func (ds *datastore) Done() <-chan struct{} {
	return ds.donech
}

func (ds *datastore) Shutdown() {
	ds.closeAll()
}

func (ds *datastore) waitReadyAll() {
	for _, c := range ds.controllers() {
		select {
		case <-c.Done():
			return
		case <-c.Ready():
		}
	}
	close(ds.readych)
}

func (ds *datastore) closeAll() {
	for _, c := range ds.controllers() {
		c.Close()
	}
}

func (ds *datastore) waitDoneAll() {
	defer close(ds.donech)
	for _, c := range ds.controllers() {
		<-c.Done()
	}
}

func (ds *datastore) controllers() []cacheController {

	potential := []cacheController{
		ds.podBase,
		ds.servicesBase,
		ds.nodesBase,
		ds.pods,
		ds.services,
		ds.nodes,
	}

	var existing []cacheController
	for _, c := range potential {
		if c != nil {
			existing = append(existing, c)
		}
	}
	return existing
}

type cacheController interface {
	Close()
	Done() <-chan struct{}
	Ready() <-chan struct{}
}
