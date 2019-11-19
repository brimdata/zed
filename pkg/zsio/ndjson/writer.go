package ndjson

import (
	"encoding/json"
	"io"

	"github.com/mccanne/zq/pkg/zson"
)

// Writer implements a Formatter for ndjson
type Writer struct {
	zson.Writer
	encoder *json.Encoder
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		Writer:  zson.Writer{w},
		encoder: json.NewEncoder(w),
	}
}

func (w *Writer) Write(rec *zson.Record) error {
	return w.encoder.Encode(rec)
}
