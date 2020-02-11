package zeekio

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/zng"
)

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
	typ = strings.ReplaceAll(typ, "enum", "zenum")
	typ = strings.ReplaceAll(typ, "vector", "array")
	return typ
}

func zngTypeToZeek(typ zng.Type) string {
	switch typ := typ.(type) {
	case *zng.TypeArray:
		return fmt.Sprintf("vector[%s]", zngTypeToZeek(typ.Type))
	case *zng.TypeSet:
		return fmt.Sprintf("set[%s]", zngTypeToZeek(typ.InnerType))
	case *zng.TypeOfFloat64:
		return "double"
	case *zng.TypeOfBstring:
		return "string"
	case *zng.TypeAlias:
		if typ.Name == "zenum" {
			return "enum"
		}
		return typ.String()
	default:
		return typ.String()
	}
}
