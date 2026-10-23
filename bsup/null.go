package bsup

import (
	"io"

	"github.com/superdb/super/pkg/field"
	"golang.org/x/sync/errgroup"
)

type NullEncoder struct {
	len uint32
}

func NewNullEncoder(len uint32) *NullEncoder {
	return &NullEncoder{len}
}

func (n *NullEncoder) Metadata(bctx *Context, off uint64) (uint64, ID) {
	return off, bctx.enter(&Null{n.len})
}

func (*NullEncoder) Encode(*errgroup.Group)             {}
func (*NullEncoder) Stats(field.Path, CollectStatsFunc) {}
func (*NullEncoder) Emit(io.Writer) error               { return nil }
