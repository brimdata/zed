package api

import (
	"reflect"
	"strings"
)

// FieldByJSONName returns the struct field with the given JSON name and a
// boolean indicating whether the field was found.
func FieldByJSONName(v reflect.Value, name string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if JSONName(t.Field(i)) == name {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// JSONName returns the JSON name of the field.  It returns the empty string if
// the field is always omitted (i.e., it has a json:"-" tag).
func JSONName(s reflect.StructField) string {
	tag := s.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	if n := strings.Split(tag, ",")[0]; n != "" {
		return n
	}
	return s.Name
}
