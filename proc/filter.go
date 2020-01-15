package proc

import (
	"github.com/mccanne/zq/filter"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type Filter struct {
	Base
	filter.Filter
}

func NewFilter(c *Context, parent Proc, f filter.Filter) *Filter {
	return &Filter{Base{Context: c, Parent: parent}, f}
}

func (f *Filter) Pull() (zbuf.Batch, error) {
	batch, err := f.Get()
	if EOS(batch, err) {
		return nil, err
	}
	defer batch.Unref()
	// Now we'll a new batch with the (sub)set of reords that match.
	out := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		r := batch.Index(k)
		if f.Filter(r) {
			out = append(out, r.Keep())
		}
	}
	//XXX need to update span... this will be done when we use Drop()
	return zbuf.NewArray(out, batch.Span()), nil
}
