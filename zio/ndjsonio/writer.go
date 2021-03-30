package ndjsonio

import (
	"encoding/json"
	"io"

	"github.com/brimdata/zed/zng"
)

// Writer implements a Formatter for ndjson
type Writer struct {
	writer  io.WriteCloser
	encoder *json.Encoder
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer:  w,
		encoder: json.NewEncoder(w),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(rec *zng.Record) error {
	return w.encoder.Encode(rec)
}
