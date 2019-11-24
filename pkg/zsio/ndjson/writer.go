package ndjson

import (
	"encoding/json"
	"io"

	"github.com/mccanne/zq/pkg/zson"
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

func (w *Writer) Write(rec *zson.Record) error {
	return w.encoder.Encode(rec)
}
