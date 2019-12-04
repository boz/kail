package kail

import (
	"encoding/json"
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
	return &writerJSON{
		out: out,
		getEnc: func(o io.Writer) *json.Encoder {
			return json.NewEncoder(o)
		},
	}
}

func NewJSONPrettyWriter(out io.Writer) Writer {
	return &writerJSON{
		out: out,
		getEnc: func(o io.Writer) *json.Encoder {
			e := json.NewEncoder(o)
			e.SetIndent("", "  ")
			return e
		},
	}
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
	out    io.Writer
	getEnc func(io.Writer) *json.Encoder
}

func (w *writerJSON) Print(ev Event) error {
	return w.Fprint(w.out, ev)
}

func (w *writerJSON) Fprint(out io.Writer, ev Event) error {

	log := ev.Log()
	if sz := len(log); sz == 0 || log[sz-1] == byte('\n') {
		log = log[:sz-1]
	}

	enc := w.getEnc(out)

	data := map[string]interface{}{
		"namespace": ev.Source().Namespace(),
		"name":      ev.Source().Name(),
		"container": ev.Source().Container(),
	}

	messageMap := map[string]interface{}{}
	if err := json.Unmarshal(log, &messageMap); err != nil {
		data["message"] = string(log)
	} else {
		data["message"] = messageMap
	}

	if err := enc.Encode(data); err != nil {
		return err
	}
	return nil
}
