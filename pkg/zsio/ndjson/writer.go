package ndjson

import (
	"encoding/json"
	"io"

	"github.com/mccanne/zq/pkg/zson"
)

// NDJSON implements a Formatter for ndjson
type NDJSON struct {
	zson.Writer
	encoder *json.Encoder
}

func NewWriter(w io.WriteCloser) *NDJSON {
	return &NDJSON{
		Writer:  zson.Writer{w},
		encoder: json.NewEncoder(w),
	}
}

func (w *NDJSON) Write(rec *zson.Record) error {
	return w.encoder.Encode(rec)
}
