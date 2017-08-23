package kail

import (
	"context"
	"fmt"
	"io"

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

	log := c.log.WithComponent(
		fmt.Sprintf("monitor [%v]", source))

	m := &_monitor{
		core:    c.cs.CoreV1(),
		source:  source,
		eventch: c.eventch,
		log:     log,
		lc:      lc,
		ctx:     c.ctx,
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
}

func (m *_monitor) Shutdown() {
	m.lc.ShutdownAsync(nil)
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

	err := <-m.lc.ShutdownRequest()
	m.lc.ShutdownInitiated(err)
	cancel()

	<-donech
}

func (m *_monitor) mainloop(ctx context.Context, donech chan struct{}) {
	defer m.log.Un(m.log.Trace("mainloop"))
	defer close(donech)

	// todo: backoff handled by k8 client?

	for ctx.Err() == nil {
		err := m.readloop(ctx)
		switch {
		case err == io.EOF:
		case err == nil:
		case ctx.Err() != nil:
			m.lc.ShutdownAsync(nil)
			return
		default:
			m.lc.ShutdownAsync(err)
			return
		}
	}
}

func (m *_monitor) readloop(ctx context.Context) error {
	defer m.log.Un(m.log.Trace("readloop"))

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
		return err
	}

	defer stream.Close()

	logbuf := make([]byte, logBufsiz)
	for ctx.Err() == nil {
		nread, err := stream.Read(logbuf)

		switch {
		case err == io.EOF:
			return err
		case ctx.Err() != nil:
			return ctx.Err()
		case err != nil:
			return m.log.Err(err, "error while reading logs")
		case nread == 0:
			return io.EOF
		}

		event := newEvent(m.source, logbuf[0:nread])

		select {
		case m.eventch <- event:
		default:
			m.log.Warnf("event buffer full. dropping logs %v", nread)
		}
	}
	return nil
}
