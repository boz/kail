package kail

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

var (
	prefixColor = color.New(color.FgHiWhite, color.Bold)
)

type Writer interface {
	Print(event Event) error
	Fprint(w io.Writer, event Event) error
}

func NewWriter(out io.Writer) Writer {
	return &writer{writerRaw{out}}
}

func NewRawWriter(out io.Writer) Writer {
	return &writerRaw{out}
}

func NewJSONWriter(out io.Writer) Writer {
	return &writerJSON{out}
}

type writer struct {
	writerRaw
}

func (w *writer) Print(ev Event) error {
	return w.Fprint(w.writerRaw.out, ev)
}

func (w *writer) Fprint(out io.Writer, ev Event) error {
	prefix := w.prefix(ev)

	if _, err := prefixColor.Fprint(out, prefix); err != nil {
		return err
	}
	if _, err := prefixColor.Fprint(out, ": "); err != nil {
		return err
	}

	return w.writerRaw.Fprint(out, ev)
}

func (w *writer) prefix(ev Event) string {
	return fmt.Sprintf("%v/%v[%v]",
		ev.Source().Namespace(),
		ev.Source().Name(),
		ev.Source().Container())
}

type writerRaw struct {
	out io.Writer
}

func (w *writerRaw) Print(ev Event) error {
	return w.Fprint(w.out, ev)
}

func (w *writerRaw) Fprint(out io.Writer, ev Event) error {
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

type writerJSON struct {
	out io.Writer
}

func (w *writerJSON) Print(ev Event) error {
	return w.Fprint(w.out, ev)
}

func (w *writerJSON) Fprint(out io.Writer, ev Event) error {

	log := ev.Log()

	if sz := len(log); sz == 0 || log[sz-1] == byte('\n') {
		log = log[:sz-1]
	}

	if log[0] != '{' && log[0] != '[' {
		log = append([]byte{'"'}, log...)
		log = append(log, '"')
	}

	if _, err := fmt.Fprintf(out,
		`{"namespace":"%s","pod":"%s","container":"%s","message":%s}%s`,
		ev.Source().Namespace(),
		ev.Source().Name(),
		ev.Source().Container(),
		log,
		"\n",
	); err != nil {
		return err
	}

	return nil
}
