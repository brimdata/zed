package zjson

import (
	"encoding/json"
	"io"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zval"
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

func (w *Writer) Write(r *zson.Record) error {
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

func (w *Writer) encodeContainer(val []byte) (interface{}, error) {
	if val == nil {
		// unset containers map to JSON empty object
		v := make(map[string]interface{})
		return v, nil
	}
	// We start out with a slice that contains nothing instead of nil
	// so that an empty containers encode to JSON empty array [].
	body := make([]interface{}, 0)
	if len(val) > 0 {
		for it := zval.Iter(val); !it.Done(); {
			v, container, err := it.Next()
			if err != nil {
				return nil, err
			}
			if container {
				child, err := w.encodeContainer(v)
				if err != nil {
					return nil, err
				}
				body = append(body, child)
			} else {
				// encode nil val as JSON null since
				// zeek.Escape() returns "" for nil
				var s interface{}
				if v != nil {
					s = zeek.Escape(v)
				}
				body = append(body, s)
			}
		}
	}
	return body, nil
}
