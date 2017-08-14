package kail

import (
	"context"

	logutil "github.com/boz/go-logutil"
	"github.com/boz/kcache/filter"
	"github.com/boz/kcache/nsname"
	"github.com/boz/kcache/types/node"
	"github.com/boz/kcache/types/pod"
	"github.com/boz/kcache/types/service"
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
		ds.stopAll()
		return nil, log.Err(err, "null filter")
	}

	if sz := len(b.namespaces); sz > 0 {
		ids := make([]nsname.NSName, 0, sz)
		for _, ns := range b.namespaces {
			ids = append(ids, nsname.New(ns, ""))
		}

		ds.pods, err = ds.pods.CloneWithFilter(filter.NSName(ids...))
		if err != nil {
			ds.stopAll()
			return nil, log.Err(err, "namespace filter")
		}
	}

	if len(b.pods) != 0 {
		ds.pods, err = ds.pods.CloneWithFilter(filter.NSName(b.pods...))
		if err != nil {
			ds.stopAll()
			return nil, log.Err(err, "pods filter")
		}
	}

	if len(b.labels) != 0 {
		ds.pods, err = ds.pods.CloneWithFilter(filter.Labels(b.labels))
		if err != nil {
			ds.stopAll()
			return nil, log.Err(err, "labels filter")
		}
	}

	if len(b.services) != 0 {
		ds.servicesBase, err = service.NewController(ctx, log, cs, "")
		if err != nil {
			ds.stopAll()
			return nil, log.Err(err, "service base controller")
		}

		ds.services, err = ds.servicesBase.CloneWithFilter(filter.NSName(b.services...))
		if err != nil {
			ds.stopAll()
			return nil, log.Err(err, "service controller")
		}
	}

	if len(b.nodes) != 0 {
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

func (ds *datastore) stopAll() {
	closeController(ds.podBase)
	closeController(ds.servicesBase)
	closeController(ds.nodesBase)
	closeController(ds.pods)
	closeController(ds.services)
	closeController(ds.nodes)
}

type closeable interface {
	Close()
}

func closeController(controller closeable) {
	if controller != nil {
		controller.Close()
	}
}
