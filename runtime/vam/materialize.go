package vam

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type Materializer struct {
	parent Puller
}

var _ zbuf.Puller = (*Materializer)(nil)

func NewMaterializer(p Puller) zbuf.Puller {
	return &Materializer{
		parent: p,
	}
}

func (m *Materializer) Pull(done bool) (zbuf.Batch, error) {
	vec, err := m.parent.Pull(done)
	if vec == nil || err != nil {
		return nil, err
	}
	b := vec.NewBuilder()
	typ := vec.Type()
	builder := zcode.NewBuilder()
	var vals []zed.Value
	for {
		if !b(builder) {
			return zbuf.NewArray(vals), nil
		}
		//XXX Body should only be called for container types, but this
		// will change soon anyway when we change out vector Builder to a
		// slot-based indexing approach that isn't based on closures.
		val := zed.NewValue(typ, bytes.Clone(builder.Bytes().Body()))
		vals = append(vals, val)
		builder.Reset()
	}
}
