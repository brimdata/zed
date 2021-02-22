package zjsonio

import (
	"errors"

	"github.com/brimsec/zq/pkg/joe"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

func encodeTypeAny(in zng.Type) joe.Interface {
	if !zng.IsContainerType(in) {
		return joe.String(tzngio.TypeString(in))
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
		return joe.String("set"), encodeTypeAny(typ.Type)
	case *zng.TypeUnion:
		return joe.String("union"), encodeTypes(typ.Types)
	case *zng.TypeEnum:
		return joe.String("enum"), encodeTypeEnum(typ)
	case *zng.TypeMap:
		types := []zng.Type{typ.KeyType, typ.ValType}
		return joe.String("map"), encodeTypes(types)
	default:
		return joe.String(tzngio.TypeString(typ)), nil
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

func encodeTypeEnum(typ *zng.TypeEnum) joe.Array {
	var out joe.Array
	out = append(out, encodeTypeObj(typ.Type))
	for _, elem := range typ.Elements {
		object := joe.NewObject()
		object["name"] = joe.String(elem.Name)
		var val interface{}
		if zng.IsContainerType(typ) {
			val, _ = encodeContainer(typ.Type, elem.Value)
		} else {
			val, _ = encodePrimitive(typ.Type, elem.Value)
		}
		object["value"] = joe.Convert(val)
		out = append(out, object)
	}
	return out
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
	case "enum":
		return decodeTypeEnum(zctx, of)
	case "map":
		return decodeTypeMap(zctx, of)
	default:
		t, err := zctx.LookupByName(string(typ))
		if err != nil {
			return nil, errors.New("zjson unknown type: " + string(typ))
		}
		return t, nil
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
		return nil, errors.New("zjson union type not an array")
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

func decodeTypeMap(zctx *resolver.Context, of joe.Interface) (*zng.TypeMap, error) {
	items, ok := of.(joe.Array)
	if !ok {
		return nil, errors.New("zjson map type not an array")
	}
	if len(items) != 2 {
		return nil, errors.New("zjson map type not an array of length 2")
	}
	keyType, err := decodeTypeAny(zctx, items[0])
	if err != nil {
		return nil, err
	}
	valType, err := decodeTypeAny(zctx, items[1])
	if err != nil {
		return nil, err
	}
	return zctx.LookupTypeMap(keyType, valType), nil
}

func decodeTypeEnum(zctx *resolver.Context, of joe.Interface) (*zng.TypeEnum, error) {
	items, ok := of.(joe.Array)
	if !ok {
		return nil, errors.New("zjson enum type not an array")
	}
	if len(items) < 2 {
		return nil, errors.New("zjson enum type array too small")
	}
	typ, err := decodeTypeAny(zctx, items[0])
	if err != nil {
		return nil, err
	}
	var elems []zng.Element
	for _, item := range items[1:] {
		obj, ok := item.(joe.Object)
		if !ok {
			return nil, errors.New("zjson enum element is not a JSON object")
		}
		elem, err := decodeEnumElement(typ, obj)
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
	return zctx.LookupTypeEnum(typ, elems), nil
}

func decodeEnumElement(typ zng.Type, object joe.Object) (zng.Element, error) {
	name, ok := object["name"]
	if !ok {
		return zng.Element{}, errors.New("zjson enum object has no name field")
	}
	sname, err := name.String()
	if err != nil {
		return zng.Element{}, errors.New("zjson enum object name field is not a string")
	}
	val, ok := object["value"]
	if !ok {
		return zng.Element{}, errors.New("zjson enum object has no value field")
	}
	var b zcode.Builder
	if err := decodeAny(&b, typ, joe.Unpack(val)); err != nil {
		return zng.Element{}, err
	}
	it := b.Bytes().Iter()
	zv, _, _ := it.Next()
	return zng.Element{sname, zv}, nil
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
		t, err := zctx.LookupByName(string(s))
		if err != nil {
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
