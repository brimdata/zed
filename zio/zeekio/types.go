package zeekio

import (
	"strings"
)

func zeekTypeToZng(typ string) string {
	// As zng types diverge from zeek types, we'll probably want to
	// re-do this but lets keep it simple for now.
	typ = strings.ReplaceAll(typ, "string", "bstring")
	return strings.ReplaceAll(typ, "vector", "array")
}

func zngTypeToZeek(typ string) string {
	typ = strings.ReplaceAll(typ, "bstring", "string")
	return strings.ReplaceAll(typ, "array", "vector")
}
