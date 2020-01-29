package zeekio

import (
	"fmt"
	"strings"

	"github.com/mccanne/zq/zng"
)

func zeekTypeToZng(typ string) string {
	// As zng types diverge from zeek types, we'll probably want to
	// re-do this but lets keep it simple for now.
	typ = strings.ReplaceAll(typ, "string", "bstring")
	return strings.ReplaceAll(typ, "vector", "array")
}

type typeVector struct {
	inner fmt.Stringer
}

func (v typeVector) String() string {
	return fmt.Sprintf("vector[%s]", v.inner)
}

func zngTypeToZeek(typ zng.Type) fmt.Stringer {
	switch typ := typ.(type) {
	case *zng.TypeArray:
		return typeVector{
			inner: zngTypeToZeek(typ.Type),
		}
	}
	switch typ {
	case zng.TypeBstring:
		return zng.TypeString
	default:
		return typ
	}
}
