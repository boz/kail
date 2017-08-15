package kail

import (
	"context"
	"fmt"
	"io"
	"sync"

	"k8s.io/api/core/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	lifecycle "github.com/boz/go-lifecycle"
	logutil "github.com/boz/go-logutil"
)

const (
	logBufsiz = 1024
)

type monitor interface {
	Shutdown()
	Done() <-chan struct{}
}

func newMonitor(c *controller, source EventSource) monitor {
	lc := lifecycle.New()
	go lc.WatchContext(c.ctx)

	log := c.log.WithComponent(fmt.Sprintf("monitor %v/%v:%v", source.Namespace(), source.Name(), source.Container()))

	m := &_monitor{
		core:    c.cs.CoreV1(),
		source:  source,
		eventch: c.eventch,
		log:     log,
		lc:      lc,
		ctx:     c.ctx,
		wg:      &sync.WaitGroup{},
	}

	go m.run()

	return m
}

type _monitor struct {
	core    corev1.CoreV1Interface
	source  EventSource
	eventch chan<- Event
	log     logutil.Log
	lc      lifecycle.Lifecycle
	ctx     context.Context
	wg      *sync.WaitGroup
}

func (m *_monitor) Shutdown() {
	m.lc.Shutdown()
}

func (m *_monitor) Done() <-chan struct{} {
	return m.lc.Done()
}

func (m *_monitor) run() {
	defer m.log.Un(m.log.Trace("run"))
	defer m.lc.ShutdownCompleted()

	ctx, cancel := context.WithCancel(m.ctx)

	donech := make(chan struct{})

	go m.mainloop(ctx, donech)

	<-m.lc.ShutdownRequest()
	m.lc.ShutdownInitiated()
	cancel()

	<-donech
}

func (m *_monitor) mainloop(ctx context.Context, donech chan struct{}) {
	defer m.log.Un(m.log.Trace("mainloop"))
	defer close(donech)
	defer m.lc.Shutdown()

	// todo: backoff

	for ctx.Err() == nil {
		err := m.readloop(ctx)
		switch err {
		case io.EOF:
		case nil:
		default:
			m.log.ErrWarn(err, "error readloop")
			return
		}
	}
}

func (m *_monitor) readloop(ctx context.Context) error {

	since := int64(1)
	opts := &v1.PodLogOptions{
		Container:    m.source.Container(),
		Follow:       true,
		SinceSeconds: &since,
	}

	req := m.core.
		Pods(m.source.Namespace()).
		GetLogs(m.source.Name(), opts)

	req = req.Context(ctx)

	stream, err := req.Stream()
	if err != nil {
		return m.log.Err(err, "error opening stream")
	}

	defer stream.Close()

	logbuf := make([]byte, logBufsiz)
	for ctx.Err() == nil {
		nread, err := stream.Read(logbuf)

		switch {
		case err == io.EOF:
			return err
		case err != nil:
			return m.log.Err(err, "error while reading logs")
		case nread == 0:
			return io.EOF
		}

		logs := string(logbuf[0:nread])
		event := newEvent(m.source, logs)

		select {
		case m.eventch <- event:
		default:
			m.log.Warnf("event buffer full. dropping logs %v", nread)
		}
	}
	return nil
}
