package zsonio

import (
	"fmt"
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
	}
	if alias, ok := typ.(*zng.TypeAlias); ok {
		name := alias.Name
		body := t.lookup(alias.Type)
		t[typ] = body
		return fmt.Sprintf("%s (=%s)", body, name)
	}
	if typ.ID() < zng.IdTypeDef {
		name := typ.String()
		t[typ] = name
		return name
	}
	return t.define(typ)
}

func (t typemap) define(typ zng.Type) string {
	if name, ok := t[typ]; ok {
		return name
	}
	switch typ := typ.(type) {
	case *zng.TypeAlias:
		panic("alias shouldn't happen in typemap.define()")
	case *zng.TypeRecord:
		var def string
		for _, col := range typ.Columns {
			if !t.known(col.Type) {
				def = t.formatRecord(typ)
				break
			}
		}
		return t.bind(typ, def)
	case *zng.TypeArray:
		def := fmt.Sprintf("[%s]", t.lookup(typ.Type))
		return t.bind(typ, def)
	case *zng.TypeSet:
		def := fmt.Sprintf("|[%s]|", t.lookup(typ.Type))
		return t.bind(typ, def)
	case *zng.TypeMap:
		def := fmt.Sprintf("|{%s,%s}|", t.lookup(typ.KeyType), t.lookup(typ.ValType))
		return t.bind(typ, def)
	case *zng.TypeUnion:
		var def string
		union := typ
		for _, typ := range union.Types {
			if !t.known(typ) {
				def = t.formatUnion(union)
				break
			}
		}
		return t.bind(typ, def)
	}
	panic("unknown case in typemap.define()")
}

func (t typemap) known(typ zng.Type) bool {
	if _, ok := t[typ]; ok {
		return true
	}
	return typ.ID() < zng.IdTypeDef
}

func (t typemap) bind(typ zng.Type, def string) string {
	id := typ.ID()
	t[typ] = strconv.Itoa(id)
	if def == "" {
		return fmt.Sprintf("=%d", id)
	}
	return fmt.Sprintf("%s (=%d)", def, id)
}

func (t typemap) lookupAlias(typ zng.Type, name string) string {
	if def, ok := t[typ]; ok {
		return def
	}
	t[typ] = name
	return "=" + name
}

func (t typemap) formatRecord(typ *zng.TypeRecord) string {
	var s strings.Builder
	var sep string
	s.WriteString("{")
	for _, col := range typ.Columns {
		s.WriteString(sep)
		s.WriteString(col.Name) // quote special chars
		s.WriteString(":")
		s.WriteString(t.lookup(col.Type))
		sep = ","
	}
	s.WriteString("}")
	return s.String()
}

func (t typemap) formatUnion(typ *zng.TypeUnion) string {
	var s strings.Builder
	sep := ""
	s.WriteString("(")
	for _, typ := range typ.Types {
		s.WriteString(sep)
		s.WriteString(t.lookup(typ))
		sep = ","
	}
	s.WriteString(")")
	return s.String()
}
