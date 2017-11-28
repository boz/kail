package kail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	source := eventSource{}

	{
		buffer := newBuffer(source)
		events := buffer.process([]byte(""))
		assert.Empty(t, events)

		events = buffer.process([]byte("foo\n"))
		assert.Len(t, events, 1)
		assert.Equal(t, "foo", string(events[0].Log()))
	}

	{
		buffer := newBuffer(source)
		events := buffer.process([]byte("foo"))
		assert.Empty(t, events)

		events = buffer.process([]byte("\n"))
		assert.Len(t, events, 1)
		assert.Equal(t, "foo", string(events[0].Log()))
	}

	{
		buffer := newBuffer(source)
		events := buffer.process([]byte("foo\n"))
		assert.Len(t, events, 1)
		assert.Equal(t, "foo", string(events[0].Log()))

		events = buffer.process([]byte("bar\n"))
		assert.Len(t, events, 1)
		assert.Equal(t, "bar", string(events[0].Log()))
	}

	{
		buffer := newBuffer(source)
		events := buffer.process([]byte("foo\nbar\n"))
		assert.Len(t, events, 2)
		assert.Equal(t, "foo", string(events[0].Log()))
		assert.Equal(t, "bar", string(events[1].Log()))

		events = buffer.process([]byte("baz\n"))
		assert.Len(t, events, 1)
		assert.Equal(t, "baz", string(events[0].Log()))
	}

	{
		buffer := newBuffer(source)
		events := buffer.process([]byte("foo\nbar"))
		assert.Len(t, events, 1)
		assert.Equal(t, "foo", string(events[0].Log()))

		events = buffer.process([]byte("baz\n"))
		assert.Len(t, events, 1)
		assert.Equal(t, "barbaz", string(events[0].Log()))
	}

}
