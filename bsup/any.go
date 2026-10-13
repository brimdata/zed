package bsup

import (
	"io"

	"github.com/superdb/super/vector"
	"golang.org/x/sync/errgroup"
)

type AnyEncoder struct {
	values   Encoder
	subtypes Encoder
}

func NewAnyEncoder(bctx *Context, vec *vector.Fusion) *AnyEncoder {
	return &AnyEncoder{
		values:   NewPrimitiveEncoder(bctx, vec.Values, false),
		subtypes: NewTypeValueEncoder(bctx, vec.Subtypes),
	}
}

func (a *AnyEncoder) Emit(w io.Writer) error {
	if err := a.values.Emit(w); err != nil {
		return err
	}
	return a.subtypes.Emit(w)
}

func (a *AnyEncoder) Encode(group *errgroup.Group) {
	a.values.Encode(group)
	a.subtypes.Encode(group)
}

func (a *AnyEncoder) Metadata(bctx *Context, off uint64) (uint64, ID) {
	off, values := a.values.Metadata(bctx, off)
	off, subtypes := a.subtypes.Metadata(bctx, off)
	return off, bctx.enter(&Any{
		Values:   values,
		Subtypes: subtypes,
	})
}
