package writers

import (
	"io"

	"github.com/boz/kail"
)

func NewRawWriter(out io.Writer) Writer {
	return &writerRaw{out}
}

type writerRaw struct {
	out io.Writer
}

func (w *writerRaw) Print(ev kail.Event) error {
	return w.Fprint(w.out, ev)
}

func (w *writerRaw) Fprint(out io.Writer, ev kail.Event) error {
	log := ev.Log()

	if _, err := out.Write(log); err != nil {
		return err
	}

	if sz := len(log); sz == 0 || log[sz-1] != byte('\n') {
		if _, err := out.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}
