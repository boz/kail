package writers

import (
	"encoding/json"
	"io"

	"github.com/boz/kail"
)

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

type writerJSON struct {
	out    io.Writer
	getEnc func(io.Writer) *json.Encoder
}

func (w *writerJSON) Print(ev kail.Event) error {
	return w.Fprint(w.out, ev)
}

func (w *writerJSON) Fprint(out io.Writer, ev kail.Event) error {

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
