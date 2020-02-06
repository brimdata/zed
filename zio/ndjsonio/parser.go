// Package ndjsonio parses ndjson records, inferring a zng type for each
// record and then parsing it into a zng value of that type.
package ndjsonio

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/buger/jsonparser"
)

type Parser struct {
	builder *zcode.Builder
	zctx    *resolver.Context
}

func NewParser(zctx *resolver.Context) *Parser {
	return &Parser{
		builder: zcode.NewBuilder(),
		zctx:    zctx,
	}
}

// Mapping json objects, numbers, strings, bools and null to the zng
// type system is straightforward. Mapping json arrays is not.
//
// We need to first traverse the array to build a list of its
// contained types. If it only contains values of a single type, then
// it is inferred as a zng array with that type as inner type.
// If it contains values of different types, then it is inferred as a
// zng array with a union type as inner type.
//
// To this end, parsing proceeds in two steps. In the first, we
// traverse the record to infer the zng type corresponding to the json
// type and record the union indexes for arrays of union types into a
// typeInfo struct.
//
// In the second, we build the zng value using the type information
// inferred in the first step.
type typeInfo interface{}
type arrayTypeInfo struct {
	tis  []typeInfo
	idxs []int
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
	ztyp, ti, err := p.jsonInferObject(val)
	if err != nil {
		panic(err)
	}
	err = p.jsonParseObject(val, ti)
	if err != nil {
		return nil, nil, err
	}
	return p.builder.Bytes(), ztyp, nil
}

func (p *Parser) jsonInferObject(b []byte) (zng.Type, typeInfo, error) {
	ti := make(map[string]interface{})
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
		return nil, nil, err
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
		var ztyp zng.Type
		var err error
		var valti typeInfo
		ztyp, valti, err = p.jsonInferValue(kv.value, kv.typ)
		if err != nil {
			return nil, nil, err
		}
		if isRecord {
			recType.Columns[nestedColno].Type = ztyp
			nestedColno += 1
			if nestedColno == len(recType.Columns) {
				nestedColno = 0
				colno += 1
			}
		} else {
			ti[columns[colno].Name] = valti
			columns[colno].Type = ztyp
			colno += 1
		}
	}
	return p.zctx.LookupTypeRecord(columns), ti, nil
}

func (p *Parser) jsonInferValue(raw []byte, typ jsonparser.ValueType) (zng.Type, typeInfo, error) {
	switch typ {
	case jsonparser.Array:
		return p.jsonInferArray(raw)
	case jsonparser.Object:
		return p.jsonInferObject(raw)
	case jsonparser.Boolean:
		return zng.TypeBool, nil, nil
	case jsonparser.Number:
		return zng.TypeDouble, nil, nil
	case jsonparser.Null, jsonparser.String:
		// XXX TypeString is no good for json null, but figuring out a better type is tricky
		return zng.TypeString, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported type %v", typ)
	}
}

func (p *Parser) jsonInferArray(raw []byte) (zng.Type, typeInfo, error) {
	var err error
	var ztyps []zng.Type
	var unionindexes []int
	var tis []typeInfo
	jsonparser.ArrayEach(raw, func(el []byte, typ jsonparser.ValueType, offset int, elErr error) {
		if err != nil {
			return
		}
		if elErr != nil {
			err = elErr
		}
		var ztyp zng.Type
		var ti typeInfo
		ztyp, ti, err = p.jsonInferValue(el, typ)
		tis = append(tis, ti)
		var found bool
		for i := range ztyps {
			if ztyps[i] == ztyp {
				unionindexes = append(unionindexes, i)
				found = true
				break
			}
		}
		if !found {
			ztyps = append(ztyps, ztyp)
			unionindexes = append(unionindexes, len(ztyps)-1)
		}
	})
	if err != nil {
		return nil, nil, err
	}
	switch len(ztyps) {
	case 0:
		return p.zctx.LookupTypeArray(zng.TypeString), arrayTypeInfo{tis, nil}, nil
	case 1:
		return p.zctx.LookupTypeArray(ztyps[0]), arrayTypeInfo{tis, nil}, nil
	default:
		return p.zctx.LookupTypeArray(p.zctx.LookupTypeUnion(ztyps)), arrayTypeInfo{tis, unionindexes}, nil
	}
}

func (p *Parser) jsonParseObject(b []byte, ti typeInfo) error {
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
		return err
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
	timap := ti.(map[string]interface{})
	for _, kv := range kvs {
		recType, isRecord := columns[colno].Type.(*zng.TypeRecord)
		if isRecord {
			if nestedColno == 0 {
				p.builder.BeginContainer()
			}
		}
		var ztyp zng.Type
		err := p.jsonParseValue(kv.value, kv.typ, timap[string(kv.key)])
		if err != nil {
			return err
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
	return nil
}

func (p *Parser) jsonParseUnionValue(raw []byte, index int, typ jsonparser.ValueType, ti typeInfo) error {
	p.builder.BeginContainer()
	defer p.builder.EndContainer()
	var a [8]byte
	n := zcode.EncodeCountedUvarint(a[:], uint64(index))
	p.builder.AppendPrimitive(a[:n])
	return p.jsonParseValue(raw, typ, ti)
}

func (p *Parser) jsonParseValue(raw []byte, typ jsonparser.ValueType, ti typeInfo) error {
	switch typ {
	case jsonparser.Array:
		p.builder.BeginContainer()
		defer p.builder.EndContainer()
		return p.jsonParseArray(raw, ti)
	case jsonparser.Object:
		p.builder.BeginContainer()
		defer p.builder.EndContainer()
		err := p.jsonParseObject(raw, ti)
		return err
	case jsonparser.Boolean:
		return p.jsonParseBool(raw)
	case jsonparser.Number:
		return p.jsonParseNumber(raw)
	case jsonparser.Null:
		return p.jsonParseNull()
	case jsonparser.String:
		return p.jsonParseString(raw)
	default:
		return fmt.Errorf("unsupported type %v", typ)
	}
}

func (p *Parser) jsonParseArray(raw []byte, ti typeInfo) error {
	var err error
	var i int
	ati := ti.(arrayTypeInfo)
	jsonparser.ArrayEach(raw, func(el []byte, typ jsonparser.ValueType, offset int, elErr error) {
		if err != nil {
			return
		}
		if elErr != nil {
			err = elErr
		}
		if ati.idxs == nil {
			err = p.jsonParseValue(el, typ, ati.tis[i])
		} else {
			err = p.jsonParseUnionValue(el, ati.idxs[i], typ, ati.tis[i])
		}
		i++
	})
	return err
}

func (p *Parser) jsonParseBool(b []byte) error {
	boolean, err := jsonparser.GetBoolean(b)
	if err != nil {
		return err
	}
	p.builder.AppendPrimitive(zng.EncodeBool(boolean))
	return nil
}

func (p *Parser) jsonParseNumber(b []byte) error {
	d, err := zng.UnsafeParseFloat64(b)
	if err != nil {
		return err
	}
	p.builder.AppendPrimitive(zng.EncodeDouble(d))
	return nil
}

func (p *Parser) jsonParseString(b []byte) error {
	b, err := jsonparser.Unescape(b, nil)
	if err != nil {
		return err
	}
	s, err := zng.TypeString.Parse(b)
	if err != nil {
		return err
	}
	p.builder.AppendPrimitive(s)
	return nil
}

func (p *Parser) jsonParseNull() error {
	p.builder.AppendPrimitive(nil)
	return nil
}
