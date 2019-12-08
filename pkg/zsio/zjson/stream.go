package zjson

import (
	"errors"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/pkg/zval"
)

type Stream struct {
	tracker *resolver.Tracker
}

func NewStream() *Stream {
	return &Stream{
		tracker: resolver.NewTracker(),
	}
}

func (s *Stream) Transform(r *zson.Record) (*Record, error) {
	id := r.Descriptor.ID
	var typ []interface{}
	if !s.tracker.Seen(id) {
		var err error
		typ, err = encodeType(r.Descriptor.Type)
		if err != nil {
			return nil, err
		}
	}
	v, err := encodeContainer(r.Raw)
	if err != nil {
		return nil, err
	}
	values, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("internal error: zson record body must be a container")
	}
	return &Record{
		Id:     id,
		Type:   typ,
		Values: values,
	}, nil
}

func encodeContainer(val []byte) (interface{}, error) {
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
				child, err := encodeContainer(v)
				if err != nil {
					return nil, err
				}
				body = append(body, child)
			} else {
				// encode nil val as JSON null since
				// zeek.Escape() returns "" for nil
				var fld interface{}
				if v != nil {
					fld = zeek.Escape(v)
				}
				body = append(body, fld)
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
func encodeType(typ *zeek.TypeRecord) ([]interface{}, error) {
	var columns []interface{}
	for _, c := range typ.Columns {
		childRec, ok := c.Type.(*zeek.TypeRecord)
		var typ interface{}
		if ok {
			var err error
			typ, err = encodeType(childRec)
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
