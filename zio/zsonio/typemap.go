package zsonio

import (
	"strconv"
	"strings"

	"github.com/brimsec/zq/zng"
)

type typemap map[zng.Type]string

func (t typemap) enter(typ zng.Type, name string) {
	t[typ] = name
}

func (t typemap) exists(typ zng.Type) bool {
	_, ok := t[typ]
	return ok
}

func (t typemap) lookup(typ zng.Type) string {
	if name, ok := t[typ]; ok {
		return name
	} else if !zng.IsContainerType(typ) {
		return typ.String()
	} else if union, ok := typ.(*zng.TypeUnion); ok {
		return t.listOf(union.Types)
	} else if typ, ok := typ.(*zng.TypeMap); ok {
		return t.listOf([]zng.Type{typ.KeyType, typ.ValType})
	}
	return strconv.Itoa(typ.ID())
}

func (t typemap) listOf(types []zng.Type) string {
	var s strings.Builder
	sep := ""
	for _, typ := range types {
		s.WriteString(sep)
		s.WriteString(t.lookup(typ))
		sep = ","
	}
	return s.String()
}
