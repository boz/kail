package kail

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/yalp/jsonpath"
)

func NewJsonWriter(out io.Writer, indent bool, jsonfilter jsonpath.FilterFunc) Writer {
	return &jsonwriter{out, indent, jsonfilter}
}

type jsonwriter struct {
	out        io.Writer
	indent     bool
	jsonfilter jsonpath.FilterFunc
}

func (w *jsonwriter) Print(ev Event) error {
	return w.Fprint(w.out, ev)
}

func (w *jsonwriter) Fprint(out io.Writer, ev Event) error {
	prefix := w.prefix(ev)

	if _, err := prefixColor.Fprint(out, prefix); err != nil {
		return err
	}
	if _, err := prefixColor.Fprint(out, ": "); err != nil {
		return err
	}

	log := ev.Log()

	// Attempt to parse log as json
	var value interface{}
	if err := json.Unmarshal(log, &value); err == nil {
		if w.jsonfilter != nil {
			// Apply jsonpath filter
			value, err = w.jsonfilter(value)
			if err != nil {
				return err
			}
		}

		if s, ok := value.(string); ok {
			// If jsonpath selects a string field, format as plain text
			log = []byte(s)
		} else {
			// Format json log output
			if w.indent {
				log, err = json.MarshalIndent(value, "", "  ")
				if err != nil {
					return err
				}
			} else {
				log, err = json.Marshal(value)
				if err != nil {
					return err
				}
			}
		}
	}

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

func (w *jsonwriter) prefix(ev Event) string {
	return fmt.Sprintf("%v/%v[%v]",
		ev.Source().Namespace(),
		ev.Source().Name(),
		ev.Source().Container())
}
