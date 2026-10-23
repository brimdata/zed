package bsup

import (
	"io"

	"github.com/superdb/super/pkg/field"
	"golang.org/x/sync/errgroup"
)

type FieldEncoder struct {
	name   string
	values Encoder
}

func (f *FieldEncoder) Metadata(bctx *Context, off uint64) (uint64, Field) {
	var id ID
	off, id = f.values.Metadata(bctx, off)
	return off, Field{
		Name:   f.name,
		Values: id,
	}
}

func (f *FieldEncoder) Encode(group *errgroup.Group, count uint32) {
	f.values.Encode(group)
}

func (f *FieldEncoder) Stats(path field.Path, fn CollectStatsFunc) {
	f.values.Stats(append(path, f.name), fn)
}

func (f *FieldEncoder) Emit(w io.Writer) error {
	return f.values.Emit(w)
}
