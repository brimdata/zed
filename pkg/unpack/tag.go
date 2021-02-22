package unpack

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	tagJSON   = "json"
	tagUnpack = "unpack"
	tagSep    = ","
)

var (
	ErrTag      = errors.New(`unpack tag must have form "", "<value>", "skip", "<value>,skip"`)
	ErrSkip     = errors.New("unpack skip tag can only appear once")
	ErrNeedJSON = errors.New("unpack tag cannot appear without a JSON tag")
)

func parseTag(which string, f reflect.StructField) (string, bool, []string) {
	tag, ok := f.Tag.Lookup(which)
	if !ok {
		return "", false, nil
	}
	if tag == "" {
		return "", true, nil
	}
	elems := strings.Split(tag, tagSep)
	var opts []string
	if len(elems) > 1 {
		opts = elems[1:]
	}
	return elems[0], true, opts
}

func jsonFieldName(f reflect.StructField) (string, bool) {
	s, ok, _ := parseTag(tagJSON, f)
	return s, ok
}

// structToRule looks through the tags in a struct to (1) make sure there
// are no duplicate tags since package json doesn't do this and it results
// in hard-to-debug behaviors, and (2) it finds an "unpack" tag which indicates
// which field of the struct to use as the key and what value to match for
// unmarshling into this concrete type.  If there is not exactly one unpack
// tag, then an error is returned.
// unpack="key,skip" unpack="key", unpack="", or unpack="skip"
func structToUnpackRule(typ reflect.Type) (string, string, error) {
	if typ.Kind() != reflect.Struct {
		return "", "", errors.New("cannot unpack into non-struct")
	}
	names := make(map[string]struct{})
	var unpackKey string
	var unpackVal string
	var unpackSkip bool
	for k := 0; k < typ.NumField(); k++ {
		field := typ.Field(k)
		jsonField, jsonOk, _ := parseTag(tagJSON, field)
		if jsonOk {
			if _, ok := names[jsonField]; ok {
				return "", "", fmt.Errorf("json field tag '%s' in struct type '%s' not unique", jsonField, typ.Name())
			}
			names[jsonField] = struct{}{}
		}
		unpackOpt, unpackOk, opts := parseTag(tagUnpack, typ.Field(k))
		if !unpackOk {
			continue
		}
		val, skip, err := parseUnpackOpts(unpackOpt, opts, typ.Name())
		if err != nil {
			return "", "", err
		}
		if skip {
			if unpackSkip {
				return "", "", ErrSkip
			}
			unpackSkip = true
		}
		if unpackKey != "" {
			return "", "", fmt.Errorf("unpack key appears twice (for JSON field %s and %s) ", unpackKey, jsonField)
		}
		if jsonField == "" {
			jsonField = field.Name
		}
		unpackKey = jsonField
		unpackVal = val
	}
	return unpackKey, unpackVal, nil
}

func parseUnpackOpts(unpackOpt string, opts []string, typName string) (string, bool, error) {
	if len(opts) > 1 {
		return "", false, ErrTag
	}
	var skip bool
	if len(opts) == 0 {
		if unpackOpt == "skip" {
			return "", true, nil
		}
	} else {
		if opts[0] != "skip" {
			return "", false, ErrTag
		}
		skip = true
	}
	if unpackOpt == "" {
		unpackOpt = typName
	}
	return unpackOpt, skip, nil
}
