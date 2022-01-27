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
		typ:  zctx.LookupTypeArray(zctx.LookupTypeArray(zed.TypeString)),
	}
}

func buildPath(typ *zed.TypeRecord, b *zcode.Builder, prefix []string) []string {
	var out []string
	for _, c := range typ.Columns {
		if typ, ok := zed.TypeUnder(c.Type).(*zed.TypeRecord); ok {
			buildPath(typ, b, append(prefix, c.Name))
		} else {
			b.BeginContainer()
			for _, s := range prefix {
				b.Append([]byte(s))
			}
			b.Append([]byte(c.Name))
			b.EndContainer()
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
	var b zcode.Builder
	buildPath(typ, &b, []string{})
	return ctx.NewValue(f.typ, b.Bytes())
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
