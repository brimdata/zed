package bsup

import (
	"io"

	"golang.org/x/sync/errgroup"
)

type MissingEncoder struct {
	len uint32
}

func NewMissingEncoder(len uint32) *MissingEncoder {
	return &MissingEncoder{len}
}

func (n *MissingEncoder) Metadata(bctx *Context, off uint64) (uint64, ID) {
	return off, bctx.enter(&Missing{n.len})
}

func (*MissingEncoder) Encode(group *errgroup.Group) {}
func (*MissingEncoder) Emit(io.Writer) error         { return nil }
