package jsonio

import (
	"io"

	"github.com/brimdata/zed"
)

type WriterOpts struct {
	Pretty int
}

type Writer struct {
	io.Closer
	encoder *encoder
}

func NewWriter(wc io.WriteCloser, opts WriterOpts) *Writer {
	return &Writer{
		Closer:  wc,
		encoder: newEncoder(wc, opts.Pretty),
	}
}

func (w *Writer) Write(val zed.Value) error {
	return w.encoder.encodeVal(val)
}
