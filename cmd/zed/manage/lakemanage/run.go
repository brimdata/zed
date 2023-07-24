package lakemanage

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/segmentio/ksuid"
)

type Run struct {
	extent.Span
	Compare expr.CompareFn
	Objects []*data.Object
	Size    int64
}

func NewRun(cmp expr.CompareFn) Run {
	return Run{Compare: cmp}
}

func (p Run) Overlaps(first, last *zed.Value) bool {
	if p.Span == nil {
		return false
	}
	return p.Span.Overlaps(first, last)
}

func (p *Run) Add(o *data.Object) {
	p.Objects = append(p.Objects, o)
	p.Size += o.Size
	if p.Span == nil {
		p.Span = extent.NewGeneric(o.Min, o.Max, p.Compare)
		return
	}
	p.Span.Extend(&o.Min)
	p.Span.Extend(&o.Max)
}

func (p *Run) ObjectIDs() []ksuid.KSUID {
	var ids []ksuid.KSUID
	for _, o := range p.Objects {
		ids = append(ids, o.ID)
	}
	return ids
}
