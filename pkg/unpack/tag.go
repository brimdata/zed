package unpack

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	tagName = "json"
	tagSep  = ","
)

func jsonFieldName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get(tagName)
	if tag != "" {
		s := strings.SplitN(tag, tagSep, 2)
		if len(s) > 0 && s[0] != "" {
			return s[0], true
		}
	}
	return "", false
}

func jsonCheckStruct(v interface{}) (reflect.Type, error) {
	typ := reflect.TypeOf(v)
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("cannot unpack into non-struct")
	}
	names := make(map[string]struct{})
	for k := 0; k < typ.NumField(); k++ {
		field, ok := jsonFieldName(typ.Field(k))
		if ok {
			if _, ok := names[field]; ok {
				return nil, fmt.Errorf("json field tag '%s' in struct type '%s' not unique", field, typ.Name())
			}
			names[field] = struct{}{}
		}
	}
	return typ, nil
}
