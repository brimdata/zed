package ndjson

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/buger/jsonparser"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
)

// ErrMultiTypedVector signifies that a json array was found with multiple types.
// Multiple-typed arrays are unsupported at this time. See zq#64.
var ErrMultiTypedVector = errors.New("vectors with multiple types are not supported")

type Parser struct {
	builder *zval.Builder
	scratch []byte
}

func NewParser() *Parser {
	return &Parser{builder: zval.NewBuilder()}
}

// Parse returns a zson.Encoding slice as well as an inferred zeek.Type
// from the provided JSON input. The function expects the input json to be an
// object, otherwise an error is returned.
func (p *Parser) Parse(b []byte) (zval.Encoding, zeek.Type, error) {
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
	return p.builder.Encode(), ztyp, nil
}

func (p *Parser) jsonParseObject(b []byte) (zeek.Type, error) {
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
	columns := make([]zeek.Column, len(kvs))
	for i, kv := range kvs {
		ztyp, err := p.jsonParseValue(kv.value, kv.typ)
		if err != nil {
			return nil, err
		}
		columns[i] = zeek.Column{Name: string(kv.key), Type: ztyp}
	}
	return &zeek.TypeRecord{Columns: columns}, nil
}

func (p *Parser) jsonParseValue(raw []byte, typ jsonparser.ValueType) (zeek.Type, error) {
	switch typ {
	case jsonparser.Array:
		p.builder.Begin()
		defer p.builder.End()
		return p.jsonParseArray(raw)
	case jsonparser.Object:
		p.builder.Begin()
		defer p.builder.End()
		return p.jsonParseObject(raw)
	case jsonparser.Boolean:
		return p.jsonParseBool(raw)
	case jsonparser.Number:
		return p.jsonParseNumber(raw)
	case jsonparser.Null:
		return p.jsonParseString(nil)
	case jsonparser.String:
		return p.jsonParseString(raw)
	default:
		return nil, fmt.Errorf("unsupported type %v", typ)
	}
}

func (p *Parser) jsonParseArray(raw []byte) (zeek.Type, error) {
	var err error
	var types []zeek.Type
	jsonparser.ArrayEach(raw, func(el []byte, typ jsonparser.ValueType, offset int, elErr error) {
		if err != nil {
			return
		}
		if elErr != nil {
			err = elErr
		}
		var ztyp zeek.Type
		ztyp, err = p.jsonParseValue(el, typ)
		types = append(types, ztyp)
	})
	if err != nil {
		return nil, err
	}
	if len(types) == 0 {
		return zeek.LookupVectorType(zeek.TypeString), nil
	}
	var vType zeek.Type
	for _, t := range types {
		if vType == nil {
			vType = t
		} else if vType != t {
			return nil, ErrMultiTypedVector
		}
	}
	return zeek.LookupVectorType(vType), nil
}

func (p *Parser) jsonParseBool(b []byte) (zeek.Type, error) {
	boolean, err := jsonparser.GetBoolean(b)
	if err != nil {
		return nil, err
	}
	val := strconv.AppendBool(p.scratch, boolean)
	p.builder.Append(val)
	return zeek.TypeBool, nil
}

// XXX This needs to handle scientific notation... I think.
func (p *Parser) jsonParseNumber(b []byte) (zeek.Type, error) {
	p.builder.Append(b)
	return zeek.TypeDouble, nil
}

func (p *Parser) jsonParseString(b []byte) (zeek.Type, error) {
	p.builder.Append(zeek.Unescape(b))
	return zeek.TypeString, nil
}
