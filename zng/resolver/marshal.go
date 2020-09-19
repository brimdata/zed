package resolver

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

var (
	errNotStruct = errors.New("not a struct or struct ptr")
)

type Marshaler interface {
	MarshalZNG(*Context, *zcode.Builder) (zng.Type, error)
}

func Marshal(zctx *Context, b *zcode.Builder, v interface{}) (zng.Type, error) {
	return encodeAny(zctx, b, reflect.ValueOf(v))
}

func MarshalRecord(zctx *Context, v interface{}) (*zng.Record, error) {
	var b zcode.Builder
	typ, err := Marshal(zctx, &b, v)
	if err != nil {
		return nil, err
	}
	recType, ok := typ.(*zng.TypeRecord)
	if !ok {
		return nil, errors.New("not a record")
	}
	body, err := b.Bytes().ContainerBody()
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(recType, body), nil
}

const (
	tagName = "zng"
	tagSep  = ","
)

func fieldName(f reflect.StructField) string {
	tag := f.Tag.Get(tagName)
	if tag != "" {
		s := strings.SplitN(tag, tagSep, 2)
		if len(s) > 0 && s[0] != "" {
			return s[0]
		}
	}
	return f.Name
}

func encodeAny(zctx *Context, b *zcode.Builder, v reflect.Value) (zng.Type, error) {
	switch v.Kind() {
	case reflect.Interface:
		m, ok := v.Interface().(Marshaler)
		if !ok {
			return nil, fmt.Errorf("couldn't marshal interface: %v", v)
		}
		return m.MarshalZNG(zctx, b)
	case reflect.Struct:
		return encodeRecord(zctx, b, v)
	case reflect.Slice:
		return encodeArray(zctx, b, v)
	case reflect.Ptr:
		if v.IsNil() {
			return encodeNil(zctx, b, v.Type())
		}
		return encodeAny(zctx, b, v.Elem())
	case reflect.String:
		b.AppendPrimitive(zng.EncodeString(v.String()))
		return zng.TypeString, nil
	case reflect.Bool:
		b.AppendPrimitive(zng.EncodeBool(v.Bool()))
		return zng.TypeBool, nil
	// XXX Need to finish the zng int types...
	case reflect.Int, reflect.Int64:
		b.AppendPrimitive(zng.EncodeInt(v.Int()))
		return zng.TypeInt64, nil
	case reflect.Int32, reflect.Int16, reflect.Int8:
		b.AppendPrimitive(zng.EncodeInt(v.Int()))
		return zng.TypeInt32, nil
	// XXX Need to finish the zng uint types...
	case reflect.Uint, reflect.Uint64:
		b.AppendPrimitive(zng.EncodeUint(v.Uint()))
		return zng.TypeUint64, nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		b.AppendPrimitive(zng.EncodeUint(v.Uint()))
		return zng.TypeUint32, nil
	// XXX add float32 to zng?
	case reflect.Float64, reflect.Float32:
		b.AppendPrimitive(zng.EncodeFloat64(v.Float()))
		return zng.TypeFloat64, nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Kind())
	}
}

func encodeNil(zctx *Context, b *zcode.Builder, t reflect.Type) (zng.Type, error) {
	typ, err := lookupType(zctx, t)
	if err != nil {
		return nil, err
	}
	if zng.IsContainerType(typ) {
		b.AppendContainer(nil)
	} else {
		b.AppendPrimitive(nil)
	}
	return typ, nil
}

func encodeRecord(zctx *Context, b *zcode.Builder, sval reflect.Value) (zng.Type, error) {
	b.BeginContainer()
	var columns []zng.Column
	stype := sval.Type()
	for i := 0; i < stype.NumField(); i++ {
		field := stype.Field(i)
		name := fieldName(field)
		typ, err := encodeAny(zctx, b, sval.Field(i))
		if err != nil {
			return nil, err
		}
		columns = append(columns, zng.Column{name, typ})
	}
	b.EndContainer()
	return zctx.LookupTypeRecord(columns)
}

func encodeArray(zctx *Context, b *zcode.Builder, arrayVal reflect.Value) (zng.Type, error) {
	len := arrayVal.Len()
	b.BeginContainer()
	var innerType zng.Type
	for i := 0; i < len; i++ {
		item := arrayVal.Index(i)
		typ, err := encodeAny(zctx, b, item)
		if err != nil {
			return nil, err
		}
		innerType = typ
	}
	b.EndContainer()
	if innerType == nil {
		// if slice was empty, look up the type without a value
		var err error
		innerType, err = lookupType(zctx, arrayVal.Type().Elem())
		if err != nil {
			return nil, err
		}
	}
	return zctx.LookupTypeArray(innerType), nil
}

func lookupType(zctx *Context, typ reflect.Type) (zng.Type, error) {
	switch typ.Kind() {
	case reflect.Struct:
		return lookupTypeRecord(zctx, typ)
	case reflect.Slice:
		typ, err := lookupType(zctx, typ.Elem())
		if err != nil {
			return nil, err
		}
		return zctx.LookupTypeArray(typ), nil
	case reflect.Ptr:
		return lookupType(zctx, typ.Elem())
	case reflect.String:
		return zng.TypeString, nil
	case reflect.Bool:
		return zng.TypeBool, nil
	case reflect.Int, reflect.Int64:
		return zng.TypeInt64, nil
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return zng.TypeInt32, nil
	case reflect.Uint, reflect.Uint64:
		return zng.TypeUint64, nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return zng.TypeUint32, nil
	case reflect.Float64, reflect.Float32:
		return zng.TypeUint64, nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", typ.Kind())
	}
}

func lookupTypeRecord(zctx *Context, structType reflect.Type) (zng.Type, error) {
	var columns []zng.Column
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		name := fieldName(field)
		fieldType, err := lookupType(zctx, field.Type)
		if err != nil {
			return nil, err
		}
		columns = append(columns, zng.Column{name, fieldType})
	}
	return zctx.LookupTypeRecord(columns)
}
