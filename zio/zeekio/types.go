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

type typeContainer struct {
	label string
	inner fmt.Stringer
}

func (v typeContainer) String() string {
	return fmt.Sprintf("%s[%s]", v.label, v.inner.String())
}

func zngTypeToZeek(typ zng.Type) fmt.Stringer {
	switch typ := typ.(type) {
	case *zng.TypeArray:
		return typeContainer{"vector", zngTypeToZeek(typ.Type)}
	case *zng.TypeSet:
		return typeContainer{"set", zngTypeToZeek(typ.InnerType)}
	case *zng.TypeOfBstring:
		return zng.TypeString
	default:
		return typ
	}
}
