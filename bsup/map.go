package bsup

import (
	"io"

	"github.com/superdb/super/pkg/field"
	"github.com/superdb/super/vector"
	"golang.org/x/sync/errgroup"
)

type MapEncoder struct {
	keys    Encoder
	values  Encoder
	offsets *Uint32Encoder
	count   uint32
}

func NewMapEncoder(bctx *Context, vec *vector.Map) *MapEncoder {
	return &MapEncoder{
		keys:    NewEncoder(bctx, vec.Keys),
		values:  NewEncoder(bctx, vec.Values),
		offsets: NewUint32Encoder(vec.Offsets),
	}
}

func (m *MapEncoder) Emit(w io.Writer) error {
	if err := m.offsets.Emit(w); err != nil {
		return err
	}
	if err := m.keys.Emit(w); err != nil {
		return err
	}
	return m.values.Emit(w)
}

func (m *MapEncoder) Metadata(bctx *Context, off uint64) (uint64, ID) {
	off, lens := m.offsets.Segment(off)
	off, keys := m.keys.Metadata(bctx, off)
	off, vals := m.values.Metadata(bctx, off)
	return off, bctx.enter(&Map{
		Lengths: lens,
		Keys:    keys,
		Values:  vals,
		Length:  m.count,
	})
}

func (*MapEncoder) Stats(field.Path, CollectStatsFunc) {} // Map stats are ignored for now.

func (m *MapEncoder) Encode(group *errgroup.Group) {
	m.offsets.Encode(group)
	m.keys.Encode(group)
	m.values.Encode(group)
}
