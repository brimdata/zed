package zsio

import (
	"io"

	"github.com/mccanne/zq/pkg/zson"
)

type Writer struct {
	zson.Writer
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{Writer: zson.Writer{w}}
}

func (w *Writer) Write(r *zson.Record) error {
	// XXX notyet
	return nil
}
