package agg

import (
	"fmt"

	"github.com/brimdata/zed"
)

type fuse struct {
	shapes   map[*zed.TypeRecord]struct{}
	partials []zed.Value
}

func newFuse() *fuse {
	return &fuse{
		shapes: make(map[*zed.TypeRecord]struct{}),
	}
}

func (f *fuse) Consume(v zed.Value) error {
	// only works for record types, e.g., fuse(foo.x) where foo.x is a record
	typ, ok := v.Type.(*zed.TypeRecord)
	if !ok {
		return nil
	}
	f.shapes[typ] = struct{}{}
	return nil
}

func (f *fuse) Result(zctx *zed.Context) (zed.Value, error) {
	if len(f.shapes)+len(f.partials) == 0 {
		// empty input
		return zed.Value{zed.TypeNull, nil}, nil
	}
	schema := NewSchema(zctx)
	for _, p := range f.partials {
		typ, err := zctx.LookupByValue(p.Bytes)
		if err != nil {
			return zed.Value{}, fmt.Errorf("invalid partial value: %w", err)
		}
		recType, ok := typ.(*zed.TypeRecord)
		if !ok {
			return zed.Value{}, fmt.Errorf("unexpected partial type %s", typ)
		}
		schema.Mixin(recType)
	}
	for typ := range f.shapes {
		schema.Mixin(typ)
	}
	typ, err := schema.Type()
	if err != nil {
		return zed.Value{}, err
	}
	return zctx.LookupTypeValue(typ), nil
}

func (f *fuse) ConsumeAsPartial(p zed.Value) error {
	if p.Type != zed.TypeType {
		return ErrBadValue
	}
	f.partials = append(f.partials, p)
	return nil
}

func (f *fuse) ResultAsPartial(zctx *zed.Context) (zed.Value, error) {
	return f.Result(zctx)
}
