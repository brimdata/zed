package vam

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type Materializer struct {
	zctx   *zed.Context
	parent vector.Puller
}

var _ zbuf.Puller = (*Materializer)(nil)

func NewMaterializer(zctx *zed.Context, p vector.Puller) zbuf.Puller {
	return &Materializer{
		zctx:   zctx,
		parent: p,
	}
}

func (m *Materializer) Pull(done bool) (zbuf.Batch, error) {
	vec, err := m.parent.Pull(done)
	if vec == nil || err != nil {
		return nil, err
	}
	variant, _ := vec.(*vector.Variant)
	var typ zed.Type
	if variant == nil {
		typ = vec.Type()
	}
	arena := zed.NewArena(m.zctx)
	defer arena.Unref()
	builder := zcode.NewBuilder()
	var vals []zed.Value
	n := vec.Len()
	for slot := uint32(0); slot < n; slot++ {
		vec.Serialize(builder, slot)
		if variant != nil {
			typ = variant.TypeOf(slot)
		}
		val := arena.NewValue(typ, bytes.Clone(builder.Bytes().Body()))
		vals = append(vals, val)
		builder.Reset()
	}
	return zbuf.NewArray(arena, vals), nil
}
