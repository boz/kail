package writers

import (
	"io"

	"github.com/boz/kail"
	"github.com/fatih/color"
)

var (
	prefixColor = color.New(color.FgHiWhite, color.Bold)
)

type Writer interface {
	Print(event kail.Event) error
	Fprint(w io.Writer, event kail.Event) error
}
