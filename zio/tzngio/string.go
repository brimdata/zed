package tzngio

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

func columnString(c zed.Column) string {
	return FormatName(c.Name) + ":" + TypeString(c.Type)
}

func ColumnString(prefix string, columns []zed.Column, suffix string) string {
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

func StringTypeEnum(t *zed.TypeEnum) string {
	var out []string
	for _, s := range t.Symbols {
		out = append(out, FormatName(s))
	}
	return fmt.Sprintf("enum[%s]", strings.Join(out, ","))
}

func FormatName(name string) string {
	if zed.IsIdentifier(name) {
		return name
	}
	var b strings.Builder
	b.WriteRune('[')
	b.WriteString(StringOfString(zed.TypeString, zed.EncodeString(name), OutFormatZNG, false))
	b.WriteRune(']')
	return b.String()
}

func TypeRecordString(columns []zed.Column) string {
	return ColumnString("record[", columns, "]")
}

func StringRecord(t *zed.TypeRecord) string {
	return TypeRecordString(t.Columns)
}

func StringTypeUnion(t *zed.TypeUnion) string {
	var ss []string
	for _, typ := range t.Types {
		ss = append(ss, TypeString(typ))
	}
	return fmt.Sprintf("union[%s]", strings.Join(ss, ","))
}

func badZng(err error, t zed.Type, zv zcode.Bytes) string {
	return fmt.Sprintf("<ZNG-ERR type %s [%s]: %s>", t, zv, err)
}

func FormatValue(v zed.Value, fmt OutFmt) string {
	if v.Bytes == nil {
		return "-"
	}
	return StringOf(v, fmt, false)
}

func TypeString(typ zed.Type) string {
	switch typ := typ.(type) {
	case *zed.TypeAlias:
		return typ.Name
	case *zed.TypeRecord:
		return StringRecord(typ)
	case *zed.TypeArray:
		return fmt.Sprintf("array[%s]", TypeString(typ.Type))
	case *zed.TypeSet:
		return fmt.Sprintf("set[%s]", TypeString(typ.Type))
	case *zed.TypeUnion:
		return StringTypeUnion(typ)
	case *zed.TypeEnum:
		return StringTypeEnum(typ)
	case *zed.TypeMap:
		return fmt.Sprintf("map[%s,%s]", TypeString(typ.KeyType), TypeString(typ.ValType))
	default:
		return typ.String()
	}
}
