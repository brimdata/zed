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
	for _, f := range typ.Fields {
		if typ, ok := zed.TypeUnder(f.Type).(*zed.TypeRecord); ok {
			buildPath(typ, b, append(prefix, f.Name))
		} else {
			b.BeginContainer()
			for _, s := range prefix {
				b.Append([]byte(s))
			}
			b.Append([]byte(f.Name))
			b.EndContainer()
		}
	}
	return out
}

func (f *Fields) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	subjectVal := args[0]
	typ := f.recordType(subjectVal)
	if typ == nil {
		return f.zctx.Missing()
	}
	//XXX should have a way to append into allocator
	var b zcode.Builder
	buildPath(typ, &b, nil)
	return ctx.NewValue(f.typ, b.Bytes())
}

func (f *Fields) recordType(val zed.Value) *zed.TypeRecord {
	if typ, ok := zed.TypeUnder(val.Type).(*zed.TypeRecord); ok {
		return typ
	}
	if val.Type == zed.TypeType {
		typ, err := f.zctx.LookupByValue(val.Bytes())
		if err != nil {
			return nil
		}
		if typ, ok := zed.TypeUnder(typ).(*zed.TypeRecord); ok {
			return typ
		}
	}
	return nil
}
