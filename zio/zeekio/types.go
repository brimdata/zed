package zeekio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zng"
)

var ErrIncompatibleZeekType = errors.New("type cannot be represented in zeek format")

// The functions defined in this file handle mappings between legacy
// Zeek types and equivalent ZNG types.
// The function zeekTypeToZng() is used when reading Zeek logs to rewrite
// types before looking up the proper Zeek type.  zngTypeToZeek() is used
// when writing Zeek logs, it should always be the inverse of zeekTypeToZng().

func isValidInputType(typ zng.Type) bool {
	switch t := typ.(type) {
	case *zng.TypeRecord, *zng.TypeUnion:
		return false
	case *zng.TypeSet:
		return isValidInputType(t.Type)
	case *zng.TypeArray:
		return isValidInputType(t.Type)
	default:
		return true
	}
}

func zeekTypeToZng(typstr string, types *tzngio.TypeParser) (zng.Type, error) {
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

func zngTypeToZeek(typ zng.Type) (string, error) {
	switch typ := typ.(type) {
	case *zng.TypeArray:
		inner, err := zngTypeToZeek(typ.Type)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("vector[%s]", inner), nil
	case *zng.TypeSet:
		inner, err := zngTypeToZeek(typ.Type)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("set[%s]", inner), nil
	case *zng.TypeOfUint8, *zng.TypeOfInt8, *zng.TypeOfInt16, *zng.TypeOfInt32, *zng.TypeOfInt64, *zng.TypeOfUint16, *zng.TypeOfUint32:
		return "int", nil
	case *zng.TypeOfUint64:
		return "count", nil
	case *zng.TypeOfFloat64:
		return "double", nil
	case *zng.TypeOfIP:
		return "addr", nil
	case *zng.TypeOfNet:
		return "subnet", nil
	case *zng.TypeOfDuration:
		return "interval", nil
	case *zng.TypeOfBstring:
		return "string", nil
	case *zng.TypeAlias:
		if typ.Name == "zenum" {
			return "enum", nil
		}
		if typ.Name == "port" {
			return "port", nil
		}
		return zngTypeToZeek(typ.Type)
	case *zng.TypeOfBool, *zng.TypeOfString, *zng.TypeOfTime:
		return typ.String(), nil
	default:
		return "", fmt.Errorf("type %s: %w", typ, ErrIncompatibleZeekType)
	}
}
