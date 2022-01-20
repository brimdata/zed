package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#fields
type Fields struct {
	zctx *zed.Context
	typ  zed.Type
}

func NewFields(zctx *zed.Context) *Fields {
	return &Fields{
		zctx: zctx,
		typ:  zctx.LookupTypeArray(zed.TypeString),
	}
}

func fieldNames(typ *zed.TypeRecord) []string {
	var out []string
	for _, c := range typ.Columns {
		if typ, ok := zed.TypeUnder(c.Type).(*zed.TypeRecord); ok {
			for _, subfield := range fieldNames(typ) {
				out = append(out, c.Name+"."+subfield)
			}
		} else {
			out = append(out, c.Name)
		}
	}
	return out
}

func (f *Fields) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zvSubject := args[0]
	typ := isRecordType(zvSubject, f.zctx)
	if typ == nil {
		return f.zctx.Missing()
	}
	//XXX should have a way to append into allocator
	var bytes zcode.Bytes
	for _, field := range fieldNames(typ) {
		bytes = zcode.Append(bytes, zcode.Bytes(field))
	}
	return ctx.NewValue(f.typ, bytes)
}

func isRecordType(zv zed.Value, zctx *zed.Context) *zed.TypeRecord {
	if typ, ok := zed.TypeUnder(zv.Type).(*zed.TypeRecord); ok {
		return typ
	}
	if zv.Type == zed.TypeType {
		typ, err := zctx.LookupByValue(zv.Bytes)
		if err != nil {
			return nil
		}
		if typ, ok := zed.TypeUnder(typ).(*zed.TypeRecord); ok {
			return typ
		}
	}
	return nil
}
