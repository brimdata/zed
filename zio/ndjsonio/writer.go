package ndjsonio

import (
	"encoding/json"
	"io"

	"github.com/brimsec/zq/zng"
)

// Writer implements a Formatter for ndjson
type Writer struct {
	io.Writer
	encoder *json.Encoder
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:  w,
		encoder: json.NewEncoder(w),
	}
}

func (w *Writer) Write(rec *zng.Record) error {
	return w.encoder.Encode(rec)
}
