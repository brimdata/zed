package bsup

import (
	"io"

	"github.com/superdb/super/pkg/field"
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

func (*MissingEncoder) Encode(group *errgroup.Group)       {}
func (*MissingEncoder) Emit(io.Writer) error               { return nil }
func (*MissingEncoder) Stats(field.Path, CollectStatsFunc) {}
