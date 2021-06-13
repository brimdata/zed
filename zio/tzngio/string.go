package tzngio

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
)

func columnString(c zng.Column) string {
	return FormatName(c.Name) + ":" + TypeString(c.Type)
}

func ColumnString(prefix string, columns []zng.Column, suffix string) string {
	var s strings.Builder
	s.WriteString(prefix)
	var comma bool
	for _, c := range columns {
		if comma {
			s.WriteByte(byte(','))
		}
		s.WriteString(columnString(c))
		comma = true
	}
	s.WriteString(suffix)
	return s.String()
}

func StringTypeEnum(t *zng.TypeEnum) string {
	typ := t.Type
	var out []string
	for _, e := range t.Elements {
		name := FormatName(e.Name)
		val := StringOf(zng.Value{typ, e.Value}, OutFormatZNG, false)
		out = append(out, fmt.Sprintf("%s:[%s]", name, val))
	}
	return fmt.Sprintf("enum[%s,%s]", typ, strings.Join(out, ","))
}

func FormatName(name string) string {
	if zng.IsIdentifier(name) {
		return name
	}
	var b strings.Builder
	b.WriteRune('[')
	b.WriteString(StringOfString(zng.TypeString, zng.EncodeString(name), OutFormatZNG, false))
	b.WriteRune(']')
	return b.String()
}

func TypeRecordString(columns []zng.Column) string {
	return ColumnString("record[", columns, "]")
}

func StringRecord(t *zng.TypeRecord) string {
	return TypeRecordString(t.Columns)
}

func StringTypeUnion(t *zng.TypeUnion) string {
	var ss []string
	for _, typ := range t.Types {
		ss = append(ss, TypeString(typ))
	}
	return fmt.Sprintf("union[%s]", strings.Join(ss, ","))
}

func badZng(err error, t zng.Type, zv zcode.Bytes) string {
	return fmt.Sprintf("<ZNG-ERR type %s [%s]: %s>", t, zv, err)
}

func FormatValue(v zng.Value, fmt OutFmt) string {
	if v.Bytes == nil {
		return "-"
	}
	return StringOf(v, fmt, false)
}

func TypeString(typ zng.Type) string {
	switch typ := typ.(type) {
	case *zng.TypeAlias:
		return typ.Name
	case *zng.TypeRecord:
		return StringRecord(typ)
	case *zng.TypeArray:
		return fmt.Sprintf("array[%s]", TypeString(typ.Type))
	case *zng.TypeSet:
		return fmt.Sprintf("set[%s]", TypeString(typ.Type))
	case *zng.TypeUnion:
		return StringTypeUnion(typ)
	case *zng.TypeEnum:
		return StringTypeEnum(typ)
	case *zng.TypeMap:
		return fmt.Sprintf("map[%s,%s]", TypeString(typ.KeyType), TypeString(typ.ValType))
	default:
		return typ.String()
	}
}
