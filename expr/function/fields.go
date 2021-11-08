package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#fields
type Fields struct {
	zctx  *zed.Context
	typ   zed.Type
	bytes zcode.Bytes
}

func fieldNames(typ *zed.TypeRecord) []string {
	var out []string
	for _, c := range typ.Columns {
		if typ, ok := zed.AliasOf(c.Type).(*zed.TypeRecord); ok {
			for _, subfield := range fieldNames(typ) {
				out = append(out, c.Name+"."+subfield)
			}
		} else {
			out = append(out, c.Name)
		}
	}
	return out
}

func (f *Fields) Call(args []zed.Value) (zed.Value, error) {
	zvSubject := args[0]
	typ := isRecordType(zvSubject, f.zctx)
	if typ == nil {
		return zed.Missing, nil
	}
	bytes := f.bytes[:0]
	for _, field := range fieldNames(typ) {
		bytes = zcode.AppendPrimitive(bytes, zcode.Bytes(field))
	}
	f.bytes = bytes
	return zed.Value{f.typ, bytes}, nil
}

func isRecordType(zv zed.Value, zctx *zed.Context) *zed.TypeRecord {
	if typ, ok := zed.AliasOf(zv.Type).(*zed.TypeRecord); ok {
		return typ
	}
	if zv.Type == zed.TypeType {
		typ, err := zctx.LookupByValue(zv.Bytes)
		if err != nil {
			return nil
		}
		if typ, ok := zed.AliasOf(typ).(*zed.TypeRecord); ok {
			return typ
		}
	}
	return nil
}
