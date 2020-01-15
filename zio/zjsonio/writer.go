package zjsonio

import (
	"encoding/json"
	"io"

	"github.com/mccanne/zq/zng"
)

type Column struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

type Record struct {
	Id     int           `json:"id"`
	Type   []interface{} `json:"type,omitempty"`
	Values []interface{} `json:"values"`
}

type Writer struct {
	io.Writer
	stream *Stream
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer: w,
		stream: NewStream(),
	}
}

func (w *Writer) Write(r *zng.Record) error {
	rec, err := w.stream.Transform(r)
	if err != nil {
		return err
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = w.Writer.Write(b)
	if err != nil {
		return err
	}
	return w.write("\n")
}

func (w *Writer) write(s string) error {
	_, err := w.Writer.Write([]byte(s))
	return err
}
