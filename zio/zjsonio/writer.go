package zjsonio

import (
	"encoding/json"
	"io"

	"github.com/brimdata/zed/pkg/joe"
	"github.com/brimdata/zed/zng"
)

type Alias struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
}

type Record struct {
	ID      int           `json:"id"`
	Type    joe.Object    `json:"schema,omitempty"`
	Aliases []Alias       `json:"aliases,omitempty"`
	Values  []interface{} `json:"values"`
}

type Writer struct {
	writer io.WriteCloser
	stream *Stream
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer: w,
		stream: NewStream(),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
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
	_, err = w.writer.Write(b)
	if err != nil {
		return err
	}
	return w.write("\n")
}

func (w *Writer) write(s string) error {
	_, err := w.writer.Write([]byte(s))
	return err
}

func (a *Alias) UnmarshalJSON(b []byte) error {
	type alias Alias
	if err := json.Unmarshal(b, (*alias)(a)); err != nil {
		return err
	}
	a.Type = joe.Convert(a.Type)
	return nil
}
