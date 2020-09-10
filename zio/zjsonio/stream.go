package zjsonio

import (
	"errors"
	"strconv"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Stream struct {
	tracker *resolver.Tracker
	aliases map[int]*zng.TypeAlias
}

func NewStream() *Stream {
	return &Stream{
		tracker: resolver.NewTracker(),
		aliases: make(map[int]*zng.TypeAlias),
	}
}

func (s *Stream) Transform(r *zng.Record) (*Record, error) {
	id := r.Type.ID()
	var typ *[]interface{}
	var aliases []Alias
	if !s.tracker.Seen(id) {
		aliases = s.encodeAliases(r.Type)
		t := encodeType(r.Type)
		typ = &t
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
		Id:      id,
		Type:    typ,
		Aliases: aliases,
		Values:  values,
	}, nil
}

func encodeUnion(typ *zng.TypeUnion, v []byte) (interface{}, error) {
	// encode nil val as JSON null since
	// zng.Escape() returns "" for nil
	if v == nil {
		return nil, nil
	}
	inner, index, v, err := typ.SplitZng(v)
	if err != nil {
		return nil, err
	}
	var fld interface{}
	if utyp, ok := (inner).(*zng.TypeUnion); ok {
		fld, err = encodeUnion(utyp, v)
	} else if zng.IsContainerType(inner) {
		fld, err = encodeContainer(inner, v)
	} else {
		fld, err = encodePrimitive(inner, v)
	}
	if err != nil {
		return nil, err
	}
	return []interface{}{strconv.Itoa(int(index)), fld}, nil
}

func encodePrimitive(typ zng.Type, v []byte) (interface{}, error) {
	// encode nil val as JSON null since
	// zng.Escape() returns "" for nil
	var fld interface{}
	if v == nil {
		return fld, nil
	}

	fieldBytes := zng.Value{typ, v}.Format(zng.OutFormatUnescaped)
	fld = string(fieldBytes)

	return fld, nil
}

func encodeContainer(typ zng.Type, val []byte) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	childType, columns := zng.ContainedType(typ)
	if childType == nil && columns == nil {
		return nil, errors.New("invalid container")
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
			childType = zng.AliasedType(childType)
			if utyp, ok := (childType).(*zng.TypeUnion); ok {
				if !container {
					return nil, zng.ErrBadValue
				}
				fld, err := encodeUnion(utyp, v)
				if err != nil {
					return nil, err
				}
				body = append(body, fld)
			} else if zng.IsContainerType(childType) {
				if !container {
					return nil, zng.ErrBadValue
				}
				child, err := encodeContainer(childType, v)
				if err != nil {
					return nil, err
				}
				body = append(body, child)
			} else {
				if container {
					return nil, zng.ErrBadValue
				}
				fld, err := encodePrimitive(childType, v)
				if err != nil {
					return nil, err
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
func encodeType(typ *zng.TypeRecord) []interface{} {
	columns := []interface{}{}
	for _, c := range typ.Columns {
		childRec, ok := c.Type.(*zng.TypeRecord)
		var typ interface{}
		if ok {
			typ = encodeType(childRec)
		} else {
			typ = c.Type.String()
		}
		columns = append(columns, Column{Name: c.Name, Type: typ})
	}
	return columns
}

func (s *Stream) encodeAliases(typ *zng.TypeRecord) []Alias {
	var aliases []Alias
	for _, alias := range zng.AliasTypes(typ) {
		id := alias.AliasID()
		if _, ok := s.aliases[id]; !ok {
			aliases = append(aliases, Alias{Name: alias.Name, Type: alias.Type.String()})
			s.aliases[id] = nil
		}
	}
	return aliases
}
