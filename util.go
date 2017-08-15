package kail

import "github.com/boz/kcache/nsname"

type EventSource interface {
	Namespace() string
	Name() string
	Container() string
	Node() string
}

type eventSource struct {
	id        nsname.NSName
	container string
	node      string
}

func (es eventSource) Namespace() string {
	return es.id.Namespace
}

func (es eventSource) Name() string {
	return es.id.Name
}

func (es eventSource) Container() string {
	return es.container
}

func (es eventSource) Node() string {
	return es.node
}

type Event interface {
	Source() EventSource
	Log() string
}

func newEvent(source EventSource, log string) Event {
	return &event{source, log}
}

type event struct {
	source EventSource
	log    string
}

func (e *event) Source() EventSource {
	return e.source
}

func (e *event) Log() string {
	return e.log
}
