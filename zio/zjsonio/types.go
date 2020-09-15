package zjsonio

import (
	"errors"

	"github.com/brimsec/zq/pkg/joe"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

func encodeTypeAny(in zng.Type) joe.Interface {
	if !zng.IsContainerType(in) {
		return joe.String(in.String())
	}
	return encodeTypeObj(in)
}

func encodeTypeObj(in zng.Type) joe.Object {
	object := joe.NewObject()
	typ, any := encodeType(in)
	object["type"] = typ
	if any != nil {
		object["of"] = any
	}
	return object
}

func encodeType(typ zng.Type) (joe.String, joe.Interface) {
	switch typ := typ.(type) {
	case *zng.TypeRecord:
		return joe.String("record"), encodeTypeColumns(typ.Columns)
	case *zng.TypeArray:
		return joe.String("array"), encodeTypeAny(typ.Type)
	case *zng.TypeSet:
		return joe.String("set"), encodeTypeAny(typ.InnerType)
	case *zng.TypeUnion:
		return joe.String("union"), encodeTypes(typ.Types)
	default:
		return joe.String(typ.String()), nil
	}
}

// Encode a type as a recursive set of JSON objects.  We could simply encode
// the top level type string, but then a javascript client would need to have
// a type parser.  Instead, we encode recursive record types as a nested set
// of objects so a javascript client can easily call JSON.parse() and have
// the record structure present in an easy-to-navigate nested object.  See the
// zjson spec for details.
func encodeTypeColumns(columns []zng.Column) joe.Array {
	var cols joe.Array
	for _, c := range columns {
		object := encodeTypeObj(c.Type)
		object["name"] = joe.String(c.Name)
		cols = append(cols, object)
	}
	return cols
}

// the record structure present in an easy-to-navigate nested object.
func encodeTypes(in []zng.Type) joe.Array {
	var types joe.Array
	for _, t := range in {
		types = append(types, encodeTypeAny(t))
	}
	return types
}

func decodeType(zctx *resolver.Context, typ joe.String, of joe.Interface) (zng.Type, error) {
	switch typ {
	default:
		t := zng.LookupPrimitive(string(typ))
		if t == nil {
			return nil, errors.New("zjson unknown type: " + string(typ))
		}
		return t, nil
	case "record":
		return decodeTypeColumns(zctx, of)
	case "set", "array":
		inner, err := decodeTypeAny(zctx, of)
		if err != nil {
			return nil, err
		}
		if typ == "array" {
			return zctx.LookupTypeArray(inner), nil
		}
		return zctx.LookupTypeSet(inner), nil
	case "union":
		return decodeTypeUnion(zctx, of)
	}
}

func decodeTypeRecord(zctx *resolver.Context, v joe.Interface) (*zng.TypeRecord, error) {
	typ, err := decodeTypeAny(zctx, v)
	if err != nil {
		return nil, err
	}
	if typ, ok := typ.(*zng.TypeRecord); ok {
		return typ, nil
	}
	return nil, errors.New("not a record type")
}

func decodeTypeColumns(zctx *resolver.Context, of joe.Interface) (*zng.TypeRecord, error) {
	cols, ok := of.(joe.Array)
	if !ok {
		return nil, errors.New("zjson record columns not an array")
	}
	var columns []zng.Column
	for _, col := range cols {
		object, ok := col.(joe.Object)
		if !ok {
			return nil, errors.New("zjson record column not an object")
		}
		typ, err := decodeTypeAny(zctx, object)
		if err != nil {
			return nil, err
		}
		name, err := object.GetString("name")
		if err != nil {
			return nil, err
		}
		column := zng.Column{
			Name: name,
			Type: typ,
		}
		columns = append(columns, column)
	}
	return zctx.LookupTypeRecord(columns)
}

func decodeTypeUnion(zctx *resolver.Context, of joe.Interface) (*zng.TypeUnion, error) {
	cols, ok := of.(joe.Array)
	if !ok {
		return nil, errors.New("zjson union types not an array")
	}
	var types []zng.Type
	for _, col := range cols {
		typ, err := decodeTypeAny(zctx, col)
		if err != nil {
			return nil, err
		}
		types = append(types, typ)
	}
	return zctx.LookupTypeUnion(types), nil
}

func decodeTypeObj(zctx *resolver.Context, in joe.Object) (zng.Type, error) {
	typ, err := in.GetString("type")
	if err != nil {
		return nil, err
	}
	of, _ := in.Get("of")
	return decodeType(zctx, joe.String(typ), of)
}

func decodeTypeAny(zctx *resolver.Context, in joe.Interface) (zng.Type, error) {
	s, ok := in.(joe.String)
	if ok {
		t := zng.LookupPrimitive(string(s))
		if t == nil {
			return nil, errors.New("zjson unknown type: " + string(s))
		}
		return t, nil
	}
	object, ok := in.(joe.Object)
	if !ok {
		return nil, errors.New("zjson type not a string or object")
	}
	typ, err := object.GetString("type")
	if err != nil {
		return nil, errors.New("zson type field inside of type object is not a string")
	}
	of, _ := in.Get("of")
	return decodeType(zctx, joe.String(typ), of)
}
