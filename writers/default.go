package writers

import (
	"fmt"
	"io"

	"github.com/boz/kail"
)

func NewWriter(out io.Writer) Writer {
	return &writer{writerRaw{out}}
}

type writer struct {
	writerRaw
}

func (w *writer) Print(ev kail.Event) error {
	return w.Fprint(w.out, ev)
}

func (w *writer) Fprint(out io.Writer, ev kail.Event) error {
	prefix := w.prefix(ev)

	if _, err := prefixColor.Fprint(out, prefix); err != nil {
		return err
	}
	if _, err := prefixColor.Fprint(out, ": "); err != nil {
		return err
	}

	return w.writerRaw.Fprint(out, ev)
}

func (w *writer) prefix(ev kail.Event) string {
	return fmt.Sprintf("%v/%v[%v]",
		ev.Source().Namespace(),
		ev.Source().Name(),
		ev.Source().Container())
}
