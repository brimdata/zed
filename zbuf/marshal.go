package zbuf

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var (
	errNotStruct = errors.New("not a struct or struct ptr")
)

type Marshaler interface {
	MarshalZNG(ctx *resolver.Context) (*zng.Record, error)
}

func Marshal(zctx *resolver.Context, v interface{}) (*zng.Record, error) {
	s, ok := getStruct(reflect.ValueOf(v))
	if !ok {
		return nil, errNotStruct
	}
	builder := zcode.NewBuilder()
	cols, err := buildStruct(zctx, builder, s)
	if err != nil {
		return nil, err
	}
	trec, err := zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	return zng.NewRecordCheck(trec, builder.Bytes())
}

func buildStruct(zctx *resolver.Context, builder *zcode.Builder, sval reflect.Value) ([]zng.Column, error) {
	var columns []zng.Column
	stype := sval.Type()
	for i := 0; i < stype.NumField(); i++ {
		field := stype.Field(i)
		name := fieldName(&field)
		value := sval.Field(i)
		if s, ok := getStruct(value); ok {
			builder.BeginContainer()
			c, err := buildStruct(zctx, builder, s)
			if err != nil {
				return nil, err
			}
			trec, err := zctx.LookupTypeRecord(c)
			if err != nil {
				return nil, err
			}
			columns = append(columns, zng.NewColumn(name, trec))
			builder.EndContainer()
		} else {
			zval, err := reflectToZngVal(&field, value)
			if err != nil {
				return nil, err
			}
			builder.AppendPrimitive(zval.Bytes)
			columns = append(columns, zng.NewColumn(name, zval.Type))
		}
	}
	return columns, nil
}

func getStruct(v reflect.Value) (reflect.Value, bool) {
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	return v, true
}

const (
	tagName = "zng"
	tagSep  = ","
)

func fieldName(f *reflect.StructField) string {
	tag := f.Tag.Get(tagName)
	if tag != "" {
		s := strings.SplitN(tag, tagSep, 2)
		if len(s) > 0 && s[0] != "" {
			return s[0]
		}
	}
	return f.Name
}

func reflectToZngVal(f *reflect.StructField, v reflect.Value) (zng.Value, error) {
	var isNil bool
	kind := f.Type.Kind()
	if kind == reflect.Ptr && v.IsNil() {
		zerov := reflect.New(f.Type.Elem())
		v = zerov.Elem()
		kind = f.Type.Elem().Kind()
		isNil = true
	}

	var zngval zng.Value
	switch kind {
	case reflect.String:
		zngval = zng.NewString(v.String())
	case reflect.Bool:
		zngval = zng.NewBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		zngval = zng.Value{
			Type:  zng.TypeInt64,
			Bytes: zng.EncodeInt(v.Int()),
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		zngval = zng.NewUint64(v.Uint())
	case reflect.Float64, reflect.Float32:
		zngval = zng.NewFloat64(v.Float())
	default:
		return zng.Value{}, fmt.Errorf("unsupported type: %v", kind)
	}

	if isNil {
		zngval.Bytes = nil
	}

	return zngval, nil
}
