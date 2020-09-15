package zjsonio

import (
	"errors"
	"strconv"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

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
