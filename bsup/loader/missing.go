package loader

import (
	"github.com/superdb/super/bsup"
	"github.com/superdb/super/pkg/field"
	"github.com/superdb/super/vector"
)

type missing struct {
	meta *bsup.Missing
}

func newMissing(meta *bsup.Missing) *missing {
	return &missing{meta: meta}
}

func (n *missing) length() uint32 {
	return n.meta.Count
}

func (*missing) unmarshal(*bsup.Context, field.Projection) {}

func (n *missing) project(loader *FrameLoader, projection field.Projection) vector.Any {
	return vector.NewMissing(n.meta.Count)
}
