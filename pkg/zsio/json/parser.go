package json

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zval"
)

// NewRawAndType returns a new zson.Raw slice as well as an inferred zeek.Type
// from provided block of JSON. The function expects the input json to be an
// object, otherwise an error is returned.
func NewRawAndType(b []byte) (zson.Raw, zeek.Type, error) {
	val, typ, _, err := jsonparser.Get(b)
	if err != nil {
		return nil, nil, err
	}
	if typ != jsonparser.Object {
		return nil, nil, fmt.Errorf("expected JSON type to be Object but got %#v", typ)
	}
	values, ztyp, err := jsonParseObject(val)
	if err != nil {
		return nil, nil, err
	}
	var raw zson.Raw
	for _, v := range values {
		raw = zval.AppendValue(raw, v)
	}
	return raw, ztyp, nil
}

func jsonParseObject(b []byte) ([][]byte, zeek.Type, error) {
	var columns []zeek.Column
	var values [][]byte
	err := jsonparser.ObjectEach(b, func(key []byte, value []byte, typ jsonparser.ValueType, offset int) error {
		val, vtyp, err := jsonParseValue(value, typ)
		if err != nil {
			return err
		}
		values = append(values, val)
		columns = append(columns, zeek.Column{Name: string(key), Type: vtyp})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	// Sort fields lexigraphically ensuring maps with the same
	// columns but different printed order get assigned the same descriptor.
	sort.Slice(columns, func(i, j int) bool {
		v := columns[i].Name < columns[j].Name
		if v {
			values[i], values[j] = values[j], values[i]
		}
		return v
	})
	return values, &zeek.TypeRecord{Columns: columns}, nil
}

func jsonParseValue(raw []byte, typ jsonparser.ValueType) ([]byte, zeek.Type, error) {
	switch typ {
	case jsonparser.Array:
		return jsonParseArray(raw)
	case jsonparser.Object:
		values, typ, err := jsonParseObject(raw)
		if err != nil {
			return nil, nil, err
		}
		return zval.AppendContainer(nil, values), typ, err
	case jsonparser.Boolean:
		return jsonParseBool(raw)
	case jsonparser.Number:
		return jsonParseNumber(raw)
	case jsonparser.Null:
		return jsonParseString(nil)
	case jsonparser.String:
		return jsonParseString(raw)
	default:
		return nil, nil, fmt.Errorf("unsupported type %v", typ)
	}
}

func jsonParseBool(raw []byte) ([]byte, zeek.Type, error) {
	b, err := jsonparser.GetBoolean(raw)
	if err != nil {
		return nil, nil, err
	}
	value := strconv.AppendBool(nil, b)
	return value, zeek.TypeBool, err
}

// XXX This needs to handle scientific notation... I think.
func jsonParseNumber(b []byte) ([]byte, zeek.Type, error) {
	if idx := bytes.IndexRune(b, '.'); idx == -1 {
		return b, zeek.TypeInt, nil
	}
	return b, zeek.TypeDouble, nil
}

// XXX do escaping guff
func jsonParseString(b []byte) ([]byte, zeek.Type, error) {
	return b, zeek.TypeString, nil
}

func jsonParseArray(raw []byte) ([]byte, zeek.Type, error) {
	var err error
	var values [][]byte
	var types []zeek.Type
	jsonparser.ArrayEach(raw, func(el []byte, typ jsonparser.ValueType, offset int, elErr error) {
		if err != nil {
			return
		}
		if elErr != nil {
			err = elErr
		}
		var val []byte
		var ztyp zeek.Type
		val, ztyp, err = jsonParseValue(el, typ)
		values = append(values, val)
		types = append(types, ztyp)
	})
	if err != nil {
		return nil, nil, err
	}
	var vType zeek.Type
	for _, t := range types {
		if vType == nil {
			vType = t
		} else if vType != t {
			vType = nil
			break
		}
	}
	// XXX zeek currently doesn't support sets with heterogeneous types so for
	// now just convert values to strings
	if vType == nil {
		vType = zeek.TypeString
	}
	return zval.AppendContainer(nil, values), zeek.LookupVectorType(vType), nil
}
