package jsonio

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/terminal/color"
	"github.com/neilotoole/jsoncolor"
)

type WriterOpts struct {
	Pretty int
}

type Writer struct {
	io.Closer
	encoder encoder
}

type encoder interface {
	Encode(any) error
}

func NewWriter(wc io.WriteCloser, opts WriterOpts) *Writer {
	var e encoder
	if opts.Pretty > 0 {
		encoder := jsoncolor.NewEncoder(wc)
		encoder.SetIndent("", strings.Repeat(" ", opts.Pretty))
		if color.Enabled {
			encoder.SetColors(jsoncolor.DefaultColors())
		}
		e = encoder
	} else {
		encoder := json.NewEncoder(wc)
		encoder.SetEscapeHTML(false)
		e = encoder
	}
	return &Writer{
		Closer:  wc,
		encoder: e,
	}
}

func (w *Writer) Write(val zed.Value) error {
	return w.encoder.Encode(marshalAny(val.Type(), val.Bytes()))
}
