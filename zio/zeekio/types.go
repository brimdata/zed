package zeekio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimsec/zq/zng"
)

var ErrIncompatibleZeekType = errors.New("type cannot be represented in zeek format")

// Compatibility between the Zeek and ZNG type systems has a few rough
// edges.  Several types have to be rewritten before we get into ZNG:
//  - Zeek "vector" is ZNG "array", since this is a container and not
//    a fully-specific type we have to rewrite it here.
//  - Zeek "string" corresponds to ZNG "bstring".  Since "string" already
//    exists in ZNG, we can't use an alias and just rewrite the name directly.
//  - Zeek "enum" corresponds to ZNG "string".  There is a desire to
//    eventually add "enum" to ZNG so we don't use an alias but rewrite
//    "enum" to "zenum" which is aliased to "string" (using the alias lets
//    us recover the original type when writing Zeek output.
//
// The function zeekTypeToZng() is used when reading Zeek logs to rewrite
// types before looking up the proper Zeek type.  zngTypeToZeek() is used
// when writing Zeek logs, it should always be the inverse of zeekTypeToZng().

func zeekTypeToZng(typ string) string {
	// As zng types diverge from zeek types, we'll probably want to
	// re-do this but lets keep it simple for now.
	typ = strings.ReplaceAll(typ, "string", "bstring")
	typ = strings.ReplaceAll(typ, "double", "float64")
	typ = strings.ReplaceAll(typ, "interval", "duration")
	typ = strings.ReplaceAll(typ, "int", "int64")
	typ = strings.ReplaceAll(typ, "count", "uint64")
	typ = strings.ReplaceAll(typ, "addr", "ip")
	typ = strings.ReplaceAll(typ, "subnet", "net")
	typ = strings.ReplaceAll(typ, "enum", "zenum")
	typ = strings.ReplaceAll(typ, "vector", "array")
	return typ
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
		inner, err := zngTypeToZeek(typ.InnerType)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("set[%s]", inner), nil
	case *zng.TypeOfByte, *zng.TypeOfInt16, *zng.TypeOfInt32, *zng.TypeOfInt64, *zng.TypeOfUint16, *zng.TypeOfUint32:
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
		return zngTypeToZeek(typ.Type)
	case *zng.TypeOfBool, *zng.TypeOfString, *zng.TypeOfPort, *zng.TypeOfTime:
		return typ.String(), nil
	default:
		return "", fmt.Errorf("type %s: %w", typ, ErrIncompatibleZeekType)
	}
}
