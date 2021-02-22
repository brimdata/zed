package zjsonio

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio/tzngio"
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

func encodeMap(typ *zng.TypeMap, v []byte) (interface{}, error) {
	// encode nil val as JSON null since
	// zng.Escape() returns "" for nil
	if v == nil {
		return nil, nil
	}
	var out []interface{}
	it := zcode.Bytes(v).Iter()
	for !it.Done() {
		key, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		v, err := encodeAny(typ.KeyType, key)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		v, err = encodeAny(typ.ValType, val)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func encodePrimitive(typ zng.Type, v []byte) (interface{}, error) {
	// encode nil val as JSON null since
	// zng.Escape() returns "" for nil
	var fld interface{}
	if v == nil {
		return fld, nil
	}

	return tzngio.StringOf(zng.Value{typ, v}, tzngio.OutFormatUnescaped, false), nil
}

func encodeAny(typ zng.Type, val []byte) (interface{}, error) {
	switch typ := typ.(type) {
	case *zng.TypeAlias:
		return encodeAny(typ.Type, val)
	case *zng.TypeUnion:
		return encodeUnion(typ, val)
	case *zng.TypeEnum:
		return encodePrimitive(zng.TypeUint64, val)
	case *zng.TypeRecord:
		return encodeRecord(typ, val)
	case *zng.TypeArray:
		return encodeContainer(typ.Type, val)
	case *zng.TypeSet:
		return encodeContainer(typ.Type, val)
	case *zng.TypeMap:
		return encodeMap(typ, val)
	default:
		return encodePrimitive(typ, val)
	}
}

func encodeRecord(typ *zng.TypeRecord, val zcode.Bytes) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	// We start out with a slice that contains nothing instead of nil
	// so that an empty container encodes as a JSON empty array [].
	out := []interface{}{}
	k := 0
	for it := val.Iter(); !it.Done(); k++ {
		zv, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		v, err := encodeAny(typ.Columns[k].Type, zv)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func encodeContainer(typ zng.Type, val zcode.Bytes) (interface{}, error) {
	if val == nil {
		return nil, nil
	}
	// We start out with a slice that contains nothing instead of nil
	// so that an empty container encodes as a JSON empty array [].
	out := []interface{}{}
	for it := val.Iter(); !it.Done(); {
		zv, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		v, err := encodeAny(typ, zv)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Stream) encodeAliases(typ *zng.TypeRecord) []Alias {
	var aliases []Alias
	for _, alias := range zng.AliasTypes(typ) {
		id := alias.AliasID()
		if _, ok := s.aliases[id]; !ok {
			v := encodeTypeAny(alias.Type)
			aliases = append(aliases, Alias{Name: alias.Name, Type: v})
			s.aliases[id] = nil
		}
	}
	return aliases
}

func decodeRecord(b *zcode.Builder, typ *zng.TypeRecord, v interface{}) error {
	values, ok := v.([]interface{})
	if !ok {
		return errors.New("zjson record value must be a JSON array")
	}
	cols := typ.Columns
	b.BeginContainer()
	for k, val := range values {
		if k >= len(cols) {
			return &zng.RecordTypeError{Name: "<record>", Type: typ.ZSON(), Err: zng.ErrExtraField}
		}
		// each column either a string value or an array of string values
		if val == nil {
			// this is an unset column
			b.AppendNull()
			continue
		}
		if err := decodeAny(b, cols[k].Type, val); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func decodePrimitive(builder *zcode.Builder, typ zng.Type, v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return errors.New("zjson primitive value is not a JSON string")
	}
	if zng.IsContainerType(typ) && !zng.IsUnionType(typ) {
		return zng.ErrNotPrimitive
	}
	zv, err := tzngio.ParseValue(typ, []byte(s))
	if err != nil {
		return err
	}
	builder.AppendPrimitive(zv)
	return nil
}

func decodeContainerBody(b *zcode.Builder, typ zng.Type, body interface{}, which string) error {
	items, ok := body.([]interface{})
	if !ok {
		return fmt.Errorf("bad json for zjson %s value", which)
	}
	for _, item := range items {
		if err := decodeAny(b, typ, item); err != nil {
			return err
		}
	}
	return nil
}

func decodeContainer(b *zcode.Builder, typ zng.Type, body interface{}, which string) error {
	if body == nil {
		b.AppendNull()
		return nil
	}
	b.BeginContainer()
	err := decodeContainerBody(b, typ, body, which)
	b.EndContainer()
	return err
}

func decodeUnion(builder *zcode.Builder, typ *zng.TypeUnion, body interface{}) error {
	if body == nil {
		builder.AppendNull()
		return nil
	}
	tuple, ok := body.([]interface{})
	if !ok {
		return errors.New("bad json for zjson union value")
	}
	if len(tuple) != 2 {
		return errors.New("zjson union value not an array of two elements")
	}
	istr, ok := tuple[0].(string)
	if !ok {
		return errors.New("bad type index for zjson union value ")
	}
	index, err := strconv.Atoi(istr)
	if err != nil {
		return fmt.Errorf("bad type index for zjson union value: %w", err)
	}
	inner, err := typ.TypeIndex(index)
	if err != nil {
		return fmt.Errorf("bad type index for zjson union value: %w", err)
	}
	builder.BeginContainer()
	builder.AppendPrimitive(zng.EncodeInt(int64(index)))
	if err := decodeAny(builder, inner, tuple[1]); err != nil {
		return err
	}
	builder.EndContainer()
	return nil
}

func decodeMap(b *zcode.Builder, typ *zng.TypeMap, body interface{}) error {
	if body == nil {
		b.AppendNull()
		return nil
	}
	items, ok := body.([]interface{})
	if !ok {
		return errors.New("bad json for zjson union value")
	}
	if len(items)&1 != 0 {
		return errors.New("zjson map value does not have an even number of elements")
	}
	b.BeginContainer()
	for k := 0; k < len(items); k += 2 {
		if err := decodeAny(b, typ.KeyType, items[k]); err != nil {
			return err
		}
		if err := decodeAny(b, typ.ValType, items[k+1]); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func decodeAny(b *zcode.Builder, typ zng.Type, body interface{}) error {
	switch typ := typ.(type) {
	case *zng.TypeAlias:
		return decodeAny(b, typ.Type, body)
	case *zng.TypeUnion:
		return decodeUnion(b, typ, body)
	case *zng.TypeMap:
		return decodeMap(b, typ, body)
	case *zng.TypeEnum:
		return decodeEnum(b, typ, body)
	case *zng.TypeRecord:
		return decodeRecord(b, typ, body)
	case *zng.TypeArray:
		err := decodeContainer(b, typ.Type, body, "array")
		return err
	case *zng.TypeSet:
		if body == nil {
			b.AppendNull()
			return nil
		}
		b.BeginContainer()
		err := decodeContainerBody(b, typ.Type, body, "set")
		b.TransformContainer(zng.NormalizeSet)
		b.EndContainer()
		return err
	default:
		return decodePrimitive(b, typ, body)
	}
}

func decodeEnum(b *zcode.Builder, typ *zng.TypeEnum, body interface{}) error {
	s, ok := body.(string)
	if !ok {
		return errors.New("zjson enum index value is not a JSON string")
	}
	index, err := strconv.Atoi(s)
	if err != nil {
		return errors.New("zjson enum index value is not a string integer")
	}
	b.AppendPrimitive(zng.EncodeUint(uint64(index)))
	return nil
}
