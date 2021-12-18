package zeekio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
)

var ErrIncompatibleZeekType = errors.New("type cannot be represented in zeek format")

// The functions defined in this file handle mappings between legacy
// Zeek types and equivalent ZNG types.
// The function zeekTypeToZNG() is used when reading Zeek logs to rewrite
// types before looking up the proper Zeek type.  zngTypeToZeek() is used
// when writing Zeek logs, it should always be the inverse of zeekTypeToZNG().

func isValidInputType(typ zed.Type) bool {
	switch t := typ.(type) {
	case *zed.TypeRecord, *zed.TypeUnion:
		return false
	case *zed.TypeSet:
		return isValidInputType(t.Type)
	case *zed.TypeArray:
		return isValidInputType(t.Type)
	default:
		return true
	}
}

func zeekTypeToZNG(typstr string, types *TypeParser) (zed.Type, error) {
	// As zng types diverge from zeek types, we'll probably want to
	// re-do this but lets keep it simple for now.
	typstr = strings.ReplaceAll(typstr, "string", "bstring")
	typstr = strings.ReplaceAll(typstr, "double", "float64")
	typstr = strings.ReplaceAll(typstr, "interval", "duration")
	typstr = strings.ReplaceAll(typstr, "int", "int64")
	typstr = strings.ReplaceAll(typstr, "count", "uint64")
	typstr = strings.ReplaceAll(typstr, "addr", "ip")
	typstr = strings.ReplaceAll(typstr, "subnet", "net")
	typstr = strings.ReplaceAll(typstr, "enum", "zenum")
	typstr = strings.ReplaceAll(typstr, "vector", "array")
	typ, err := types.Parse(typstr)
	if err != nil {
		return nil, err
	}
	if !isValidInputType(typ) {
		return nil, ErrIncompatibleZeekType
	}
	return typ, nil
}

func zngTypeToZeek(typ zed.Type) (string, error) {
	switch typ := typ.(type) {
	case *zed.TypeArray:
		inner, err := zngTypeToZeek(typ.Type)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("vector[%s]", inner), nil
	case *zed.TypeSet:
		inner, err := zngTypeToZeek(typ.Type)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("set[%s]", inner), nil
	case *zed.TypeOfUint8, *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64, *zed.TypeOfUint16, *zed.TypeOfUint32:
		return "int", nil
	case *zed.TypeOfUint64:
		return "count", nil
	case *zed.TypeOfFloat32, *zed.TypeOfFloat64:
		return "double", nil
	case *zed.TypeOfIP:
		return "addr", nil
	case *zed.TypeOfNet:
		return "subnet", nil
	case *zed.TypeOfDuration:
		return "interval", nil
	case *zed.TypeOfBstring:
		return "string", nil
	case *zed.TypeAlias:
		if typ.Name == "zenum" {
			return "enum", nil
		}
		if typ.Name == "port" {
			return "port", nil
		}
		return zngTypeToZeek(typ.Type)
	case *zed.TypeOfBool, *zed.TypeOfString, *zed.TypeOfTime:
		return typ.String(), nil
	default:
		return "", fmt.Errorf("type %s: %w", typ, ErrIncompatibleZeekType)
	}
}
