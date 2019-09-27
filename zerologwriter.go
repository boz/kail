package kail

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rs/zerolog"
)

func NewZerologWriter(out io.Writer) Writer {
	return &zerologwriter{out}
}

type zerologwriter struct {
	out io.Writer
}

func (w *zerologwriter) Print(ev Event) error {
	return w.Fprint(w.out, ev)
}

func (w *zerologwriter) Fprint(out io.Writer, ev Event) error {
	prefix := w.prefix(ev)

	if _, err := prefixColor.Fprint(out, prefix); err != nil {
		return err
	}
	if _, err := prefixColor.Fprint(out, ": "); err != nil {
		return err
	}

	log := ev.Log()

	// Attempt to parse log as json
	var v interface{}
	if err := json.Unmarshal(log, &v); err == nil {
		consoleWriter := zerolog.ConsoleWriter{Out: w.out}
		if _, err := consoleWriter.Write(log); err != nil {
			return err
		}
		return nil
	}

	// Could not parse as json, so revert to default log output
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

func (w *zerologwriter) prefix(ev Event) string {
	return fmt.Sprintf("%v/%v[%v]",
		ev.Source().Namespace(),
		ev.Source().Name(),
		ev.Source().Container())
}
