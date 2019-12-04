package zjson

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
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
	tracker *resolver.Tracker
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:  w,
		tracker: resolver.NewTracker(),
	}
}

func (w *Writer) Write(r *zson.Record) error {
	id := r.Descriptor.ID
	var typ []interface{}
	if !w.tracker.Seen(id) {
		var err error
		typ, err = EncodeType(r.Descriptor.Type)
		if err != nil {
			return err
		}
	}
	v, err := w.encodeContainer(r.Raw)
	if err != nil {
		return err
	}
	values, ok := v.([]interface{})
	if !ok {
		return errors.New("internal error: zson record body must be a container")
	}
	rec := Record{
		Id:     id,
		Type:   typ,
		Values: values,
	}
	b, err := json.Marshal(&rec)
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

// Encode a type as a resursive set of JSON objects.  We could simply encode
// the top level type string, but then a javascript client would need to have
// a type parser.  Instead, we encode recursive record types as a nested set
// of objects so a javascript client can easily call JSON.parse() and have
// the record structure present in an easy-to-navigate nested object.
func EncodeType(typ *zeek.TypeRecord) ([]interface{}, error) {
	var columns []interface{}
	for _, c := range typ.Columns {
		childRec, ok := c.Type.(*zeek.TypeRecord)
		var typ interface{}
		if ok {
			var err error
			typ, err = EncodeType(childRec)
			if err != nil {
				return nil, err
			}
		} else {
			typ = c.Type.String()
		}
		columns = append(columns, Column{Name: c.Name, Type: typ})
	}
	return columns, nil
}
