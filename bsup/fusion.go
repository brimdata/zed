package bsup

import (
	"io"

	"github.com/superdb/super/pkg/field"
	"github.com/superdb/super/vector"
	"golang.org/x/sync/errgroup"
)

type FusionEncoder struct {
	values   Encoder
	subtypes Encoder
}

var _ Encoder = (*FusionEncoder)(nil)

func NewFusionEncoder(bctx *Context, vec *vector.Fusion) *FusionEncoder {
	return &FusionEncoder{
		values: NewEncoder(bctx, vec.Values),
		// Call NewTypeValueEncoder directly because we do not want subtypes
		// wrapped in a dict / const.
		subtypes: NewTypeValueEncoder(bctx, vec.Subtypes),
	}
}

func (f *FusionEncoder) Emit(w io.Writer) error {
	if err := f.values.Emit(w); err != nil {
		return err
	}
	return f.subtypes.Emit(w)
}

func (f *FusionEncoder) Encode(group *errgroup.Group) {
	f.values.Encode(group)
	f.subtypes.Encode(group)
}

func (f *FusionEncoder) Metadata(bctx *Context, off uint64) (uint64, ID) {
	off, values := f.values.Metadata(bctx, off)
	off, subtypes := f.subtypes.Metadata(bctx, off)
	return off, bctx.enter(&Fusion{
		Values:   values,
		Subtypes: subtypes,
	})
}

func (i *FusionEncoder) Stats(path field.Path, fn CollectStatsFunc) {
	i.values.Stats(path, fn)
}
