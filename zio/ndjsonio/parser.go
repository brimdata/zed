package ndjsonio

import (
	"bytes"
	"errors"
	"fmt"
	"sort"

	"github.com/buger/jsonparser"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zio/zeekio"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

// ErrMultiTypedVector signifies that a json array was found with multiple types.
// Multiple-typed arrays are unsupported at this time. See zq#64.
var ErrMultiTypedVector = errors.New("vectors with multiple types are not supported")

type Parser struct {
	builder *zcode.Builder
	zctx    *resolver.Context
	scratch []byte
}

func NewParser(zctx *resolver.Context) *Parser {
	return &Parser{
		builder: zcode.NewBuilder(),
		zctx:    zctx,
	}
}

// Parse returns a zng.Encoding slice as well as an inferred zng.Type
// from the provided JSON input. The function expects the input json to be an
// object, otherwise an error is returned.
func (p *Parser) Parse(b []byte) (zcode.Bytes, zng.Type, error) {
	val, typ, _, err := jsonparser.Get(b)
	if err != nil {
		return nil, nil, err
	}
	if typ != jsonparser.Object {
		return nil, nil, fmt.Errorf("expected JSON type to be Object but got %#v", typ)
	}
	p.builder.Reset()
	ztyp, err := p.jsonParseObject(val)
	if err != nil {
		return nil, nil, err
	}
	return p.builder.Bytes(), ztyp, nil
}

func (p *Parser) jsonParseObject(b []byte) (zng.Type, error) {
	type kv struct {
		key   []byte
		value []byte
		typ   jsonparser.ValueType
	}
	var kvs []kv
	err := jsonparser.ObjectEach(b, func(key []byte, value []byte, typ jsonparser.ValueType, offset int) error {
		kvs = append(kvs, kv{key, value, typ})
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Sort fields lexigraphically ensuring maps with the same
	// columns but different printed order get assigned the same descriptor.
	sort.Slice(kvs, func(i, j int) bool {
		return bytes.Compare(kvs[i].key, kvs[j].key) < 0
	})

	// Build the list of columns (without types yet) and then run them
	// through Unflatten() to find nested records.
	columns := make([]zng.Column, len(kvs))
	for i, kv := range kvs {
		columns[i] = zng.NewColumn(string(kv.key), zng.TypeString)
	}
	columns, _ = zeekio.Unflatten(p.zctx, columns, false)

	// Parse the actual values and fill in column types along the way,
	// taking care to step into nested records as necessary.
	colno := 0
	nestedColno := 0
	for _, kv := range kvs {
		recType, isRecord := columns[colno].Type.(*zng.TypeRecord)
		if isRecord {
			if nestedColno == 0 {
				p.builder.BeginContainer()
			}
		}

		ztyp, err := p.jsonParseValue(kv.value, kv.typ)
		if err != nil {
			return nil, err
		}

		if isRecord {
			recType.Columns[nestedColno].Type = ztyp
			nestedColno += 1
			if nestedColno == len(recType.Columns) {
				p.builder.EndContainer()
				nestedColno = 0
				colno += 1
			}
		} else {
			columns[colno].Type = ztyp
			colno += 1
		}
	}
	return &zng.TypeRecord{Columns: columns}, nil
}

func (p *Parser) jsonParseValue(raw []byte, typ jsonparser.ValueType) (zng.Type, error) {
	switch typ {
	case jsonparser.Array:
		p.builder.BeginContainer()
		defer p.builder.EndContainer()
		return p.jsonParseArray(raw)
	case jsonparser.Object:
		p.builder.BeginContainer()
		defer p.builder.EndContainer()
		return p.jsonParseObject(raw)
	case jsonparser.Boolean:
		return p.jsonParseBool(raw)
	case jsonparser.Number:
		return p.jsonParseNumber(raw)
	case jsonparser.Null:
		return p.jsonParseNull()
	case jsonparser.String:
		return p.jsonParseString(raw)
	default:
		return nil, fmt.Errorf("unsupported type %v", typ)
	}
}

func (p *Parser) jsonParseArray(raw []byte) (zng.Type, error) {
	var err error
	var types []zng.Type
	jsonparser.ArrayEach(raw, func(el []byte, typ jsonparser.ValueType, offset int, elErr error) {
		if err != nil {
			return
		}
		if elErr != nil {
			err = elErr
		}
		var ztyp zng.Type
		ztyp, err = p.jsonParseValue(el, typ)
		types = append(types, ztyp)
	})
	if err != nil {
		return nil, err
	}
	if len(types) == 0 {
		return p.zctx.LookupTypeVector(zng.TypeString), nil
	}
	var vType zng.Type
	for _, t := range types {
		if vType == nil {
			vType = t
		} else if vType != t {
			// XXX fix this with ZNG type any
			return nil, ErrMultiTypedVector
		}
	}
	return p.zctx.LookupTypeVector(vType), nil
}

func (p *Parser) jsonParseBool(b []byte) (zng.Type, error) {
	boolean, err := jsonparser.GetBoolean(b)
	if err != nil {
		return nil, err
	}
	p.builder.AppendPrimitive(zng.EncodeBool(boolean))
	return zng.TypeBool, nil
}

func (p *Parser) jsonParseNumber(b []byte) (zng.Type, error) {
	d, err := zng.UnsafeParseFloat64(b)
	if err != nil {
		return nil, err
	}
	p.builder.AppendPrimitive(zng.EncodeDouble(d))
	return zng.TypeDouble, nil
}

func (p *Parser) jsonParseString(b []byte) (zng.Type, error) {
	b, err := jsonparser.Unescape(b, nil)
	if err != nil {
		return nil, err
	}
	s, err := zng.TypeString.Parse(b)
	if err != nil {
		return nil, err
	}
	p.builder.AppendPrimitive(s)
	return zng.TypeString, nil
}

func (p *Parser) jsonParseNull() (zng.Type, error) {
	p.builder.AppendPrimitive(nil)
	// XXX TypeString is no good but figuring out a better type is tricky
	return zng.TypeString, nil
}
