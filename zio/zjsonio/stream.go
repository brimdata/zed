package zjsonio

import (
	"errors"

	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Stream struct {
	tracker *resolver.Tracker
}

func NewStream() *Stream {
	return &Stream{
		tracker: resolver.NewTracker(),
	}
}

func (s *Stream) Transform(r *zng.Record) (*Record, error) {
	id := r.Type.ID()
	var typ []interface{}
	if !s.tracker.Seen(id) {
		var err error
		typ, err = encodeType(r.Type)
		if err != nil {
			return nil, err
		}
	}
	v, err := encodeContainer(r.Type, r.Raw)
	if err != nil {
		return nil, err
	}
	values, ok := v.([]interface{})
	if !ok {
		return nil, errors.New("internal error: zng record body must be a container")
	}
	return &Record{
		Id:     id,
		Type:   typ,
		Values: values,
	}, nil
}

func encodeContainer(typ zng.Type, val []byte) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	childType, columns := zng.ContainedType(typ)
	if childType == nil && columns == nil {
		return nil, zbuf.ErrSyntax
	}
	k := 0
	// We start out with a slice that contains nothing instead of nil
	// so that an empty containers encode to JSON empty array [].
	body := make([]interface{}, 0)
	if len(val) > 0 {
		for it := zcode.Iter(val); !it.Done(); {
			v, container, err := it.Next()
			if err != nil {
				return nil, err
			}
			if columns != nil {
				if k >= len(columns) {
					return nil, &zng.RecordTypeError{Name: "<record>", Type: typ.String(), Err: zng.ErrExtraField}
				}
				childType = columns[k].Type
				k++
			}
			if container {
				child, err := encodeContainer(childType, v)
				if err != nil {
					return nil, err
				}
				body = append(body, child)
			} else {
				// encode nil val as JSON null since
				// zng.Escape() returns "" for nil
				var fld interface{}
				if v != nil {
					fieldBytes := zng.Value{childType, v}.Format(zng.OutFormatUnescaped)
					fld = string(fieldBytes)
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
func encodeType(typ *zng.TypeRecord) ([]interface{}, error) {
	var columns []interface{}
	for _, c := range typ.Columns {
		childRec, ok := c.Type.(*zng.TypeRecord)
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
