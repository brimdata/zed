package proc

import (
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zson"
)

type HeadProc struct {
	Base
	limit, count int
}

func NewHeadProc(c *Context, parent Proc, limit int) *HeadProc {
	return &HeadProc{Base{Context: c, Parent: parent}, limit, 0}
}

func (h *HeadProc) Pull() (zson.Batch, error) {
	remaining := h.limit - h.count
	if remaining <= 0 {
		return nil, nil
	}
	batch, err := h.Get()
	if EOS(batch, err) {
		return nil, err
	}
	n := batch.Length()
	if n < remaining {
		// This batch has fewer than the needed records.
		// Send them all downstream and update the count.
		h.count += n
		return batch, nil
	}
	defer batch.Unref()
	// This batch has more than the needed records.
	// Create a new batch and copy only the needed records.
	// Then signal to the upstream that we're done.
	recs := make([]*zson.Record, remaining)
	for k := 0; k < remaining; k++ {
		recs[k] = batch.Index(k).Keep()
	}
	h.count = h.limit
	h.Done()
	return zson.NewArray(recs, nano.NewSpanTs(h.MinTs, h.MaxTs)), nil
}
