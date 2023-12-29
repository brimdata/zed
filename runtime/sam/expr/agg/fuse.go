package agg

import (
	"fmt"

	"github.com/brimdata/zed"
)

type fuse struct {
	shapes   map[zed.Type]int
	partials []zed.Value
}

var _ Function = (*fuse)(nil)

func newFuse() *fuse {
	return &fuse{
		shapes: make(map[zed.Type]int),
	}
}

func (f *fuse) Consume(val zed.Value) {
	if _, ok := f.shapes[val.Type()]; !ok {
		f.shapes[val.Type()] = len(f.shapes)
	}
}

func (f *fuse) Result(a *zed.Arena) zed.Value {
	if len(f.shapes)+len(f.partials) == 0 {
		return zed.NullType
	}
	schema := NewSchema(a.Zctx())
	for _, p := range f.partials {
		typ, err := a.Zctx().LookupByValue(p.Bytes())
		if err != nil {
			panic(fmt.Errorf("fuse: invalid partial value: %w", err))
		}
		schema.Mixin(typ)
	}
	shapes := make([]zed.Type, len(f.shapes))
	for typ, i := range f.shapes {
		shapes[i] = typ
	}
	for _, typ := range shapes {
		schema.Mixin(typ)
	}
	return a.LookupTypeValue(schema.Type())
}

func (f *fuse) ConsumeAsPartial(partial zed.Value) {
	if partial.Type() != zed.TypeType {
		panic("fuse: partial not a type value")
	}
	f.partials = append(f.partials, partial.Copy())
}

func (f *fuse) ResultAsPartial(a *zed.Arena) zed.Value {
	return f.Result(a)
}
