package zjsonio

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

func encodeUnion(zctx *zed.Context, union *zed.TypeUnion, bytes zcode.Bytes) (interface{}, error) {
	// encode nil val as JSON null since
	// zed.Escape() returns "" for nil
	if bytes == nil {
		return nil, nil
	}
	inner, selector, b, err := union.SplitZNG(bytes)
	if err != nil {
		return nil, err
	}
	val, err := encodeValue(zctx, inner, b)
	if err != nil {
		return nil, err
	}
	return []interface{}{strconv.Itoa(int(selector)), val}, nil
}

func encodeMap(zctx *zed.Context, typ *zed.TypeMap, v zcode.Bytes) (interface{}, error) {
	// encode nil val as JSON null since
	// zed.Escape() returns "" for nil
	if v == nil {
		return nil, nil
	}
	var out []interface{}
	it := zcode.Bytes(v).Iter()
	for !it.Done() {
		pair := make([]interface{}, 2)
		var err error
		pair[0], err = encodeValue(zctx, typ.KeyType, it.Next())
		if err != nil {
			return nil, err
		}
		pair[1], err = encodeValue(zctx, typ.ValType, it.Next())
		if err != nil {
			return nil, err
		}
		out = append(out, pair)
	}
	return out, nil
}

func encodePrimitive(zctx *zed.Context, typ zed.Type, v zcode.Bytes) (interface{}, error) {
	// encode nil val as JSON null since
	// zed.Escape() returns "" for nil
	var fld interface{}
	if v == nil {
		return fld, nil
	}
	if typ == zed.TypeType {
		typ, err := zctx.LookupByValue(v)
		if err != nil {
			return nil, err
		}
		if zed.TypeID(typ) < zed.IDTypeComplex {
			return zed.PrimitiveName(typ), nil
		}
		if named, ok := typ.(*zed.TypeNamed); ok {
			return named.Name, nil
		}
		return strconv.Itoa(zed.TypeID(typ)), nil
	}
	if typ.ID() == zed.IDString {
		return string(v), nil
	}
	return zson.FormatPrimitive(typ, v), nil
}

func encodeValue(zctx *zed.Context, typ zed.Type, val zcode.Bytes) (interface{}, error) {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return encodeValue(zctx, typ.Type, val)
	case *zed.TypeUnion:
		return encodeUnion(zctx, typ, val)
	case *zed.TypeEnum:
		return encodePrimitive(zctx, zed.TypeUint64, val)
	case *zed.TypeRecord:
		return encodeRecord(zctx, typ, val)
	case *zed.TypeArray:
		return encodeContainer(zctx, typ.Type, val)
	case *zed.TypeSet:
		return encodeContainer(zctx, typ.Type, val)
	case *zed.TypeMap:
		return encodeMap(zctx, typ, val)
	case *zed.TypeError:
		return encodeValue(zctx, typ.Type, val)
	default:
		return encodePrimitive(zctx, typ, val)
	}
}

func encodeRecord(zctx *zed.Context, typ *zed.TypeRecord, val zcode.Bytes) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	// We start out with a slice that contains nothing instead of nil
	// so that an empty container encodes as a JSON empty array [].
	out := []interface{}{}
	k := 0
	for it := val.Iter(); !it.Done(); k++ {
		v, err := encodeValue(zctx, typ.Columns[k].Type, it.Next())
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func encodeContainer(zctx *zed.Context, typ zed.Type, bytes zcode.Bytes) (interface{}, error) {
	if bytes == nil {
		return nil, nil
	}
	// We start out with a slice that contains nothing instead of nil
	// so that an empty container encodes as a JSON empty array [].
	out := []interface{}{}
	for it := bytes.Iter(); !it.Done(); {
		v, err := encodeValue(zctx, typ, it.Next())
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func decodeRecord(b *zcode.Builder, typ *zed.TypeRecord, v interface{}) error {
	values, ok := v.([]interface{})
	if !ok {
		return errors.New("ZJSON record value must be a JSON array")
	}
	cols := typ.Columns
	b.BeginContainer()
	for k, val := range values {
		if k >= len(cols) {
			return zed.ErrExtraField
		}
		// each column either a string value or an array of string values
		if val == nil {
			b.Append(nil)
			continue
		}
		if err := decodeValue(b, cols[k].Type, val); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func decodePrimitive(builder *zcode.Builder, typ zed.Type, v interface{}) error {
	if zed.IsContainerType(typ) && !zed.IsUnionType(typ) {
		return zed.ErrNotPrimitive
	}
	text, ok := v.(string)
	if !ok {
		return errors.New("ZJSON primitive value is not a JSON string")
	}
	val := zson.Primitive{
		Type: typ,
		Text: text,
	}
	err := zson.BuildPrimitive(builder, val)
	return err
}

func decodeContainerBody(b *zcode.Builder, typ zed.Type, body interface{}, which string) error {
	items, ok := body.([]interface{})
	if !ok {
		return fmt.Errorf("bad json for ZJSON %s value", which)
	}
	for _, item := range items {
		if err := decodeValue(b, typ, item); err != nil {
			return err
		}
	}
	return nil
}

func decodeContainer(b *zcode.Builder, typ zed.Type, body interface{}, which string) error {
	if body == nil {
		b.Append(nil)
		return nil
	}
	b.BeginContainer()
	err := decodeContainerBody(b, typ, body, which)
	b.EndContainer()
	return err
}

func decodeUnion(builder *zcode.Builder, typ *zed.TypeUnion, body interface{}) error {
	if body == nil {
		builder.Append(nil)
		return nil
	}
	tuple, ok := body.([]interface{})
	if !ok {
		return errors.New("bad json for ZJSON union value")
	}
	if len(tuple) != 2 {
		return errors.New("ZJSON union value not an array of two elements")
	}
	selectorStr, ok := tuple[0].(string)
	if !ok {
		return errors.New("bad selector for ZJSON union value")
	}
	selector, err := strconv.Atoi(selectorStr)
	if err != nil {
		return fmt.Errorf("bad selector for ZJSON union value: %w", err)
	}
	inner, err := typ.Type(selector)
	if err != nil {
		return fmt.Errorf("bad selector for ZJSON union value: %w", err)
	}
	builder.BeginContainer()
	builder.Append(zed.EncodeInt(int64(selector)))
	if err := decodeValue(builder, inner, tuple[1]); err != nil {
		return err
	}
	builder.EndContainer()
	return nil
}

func decodeMap(b *zcode.Builder, typ *zed.TypeMap, body interface{}) error {
	if body == nil {
		b.Append(nil)
		return nil
	}
	items, ok := body.([]interface{})
	if !ok {
		return errors.New("bad json for ZJSON union value")
	}
	b.BeginContainer()
	for _, item := range items {
		pair, ok := item.([]interface{})
		if !ok || len(pair) != 2 {
			return errors.New("ZJSON map value must be an array of two-element arrays")
		}
		if err := decodeValue(b, typ.KeyType, pair[0]); err != nil {
			return err
		}
		if err := decodeValue(b, typ.ValType, pair[1]); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func decodeValue(b *zcode.Builder, typ zed.Type, body interface{}) error {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return decodeValue(b, typ.Type, body)
	case *zed.TypeUnion:
		return decodeUnion(b, typ, body)
	case *zed.TypeMap:
		return decodeMap(b, typ, body)
	case *zed.TypeEnum:
		return decodeEnum(b, typ, body)
	case *zed.TypeRecord:
		return decodeRecord(b, typ, body)
	case *zed.TypeArray:
		err := decodeContainer(b, typ.Type, body, "array")
		return err
	case *zed.TypeSet:
		if body == nil {
			b.Append(nil)
			return nil
		}
		b.BeginContainer()
		err := decodeContainerBody(b, typ.Type, body, "set")
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		return err
	case *zed.TypeError:
		return decodeValue(b, typ.Type, body)
	default:
		return decodePrimitive(b, typ, body)
	}
}

func decodeEnum(b *zcode.Builder, typ *zed.TypeEnum, body interface{}) error {
	s, ok := body.(string)
	if !ok {
		return errors.New("ZJSON enum index value is not a JSON string")
	}
	index, err := strconv.Atoi(s)
	if err != nil {
		return errors.New("ZJSON enum index value is not a string integer")
	}
	b.Append(zed.EncodeUint(uint64(index)))
	return nil
}
