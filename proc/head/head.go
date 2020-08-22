package head

import (
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Proc struct {
	proc.Parent
	limit, count int
}

func New(parent proc.Interface, limit int) *Proc {
	return &Proc{
		Parent: proc.Parent{parent},
		limit:  limit,
	}
}

func (h *Proc) Pull() (zbuf.Batch, error) {
	remaining := h.limit - h.count
	if remaining <= 0 {
		return nil, nil
	}
	batch, err := h.Parent.Pull()
	if proc.EOS(batch, err) {
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
	recs := make([]*zng.Record, remaining)
	for k := 0; k < remaining; k++ {
		recs[k] = batch.Index(k).Keep()
	}
	h.count = h.limit
	h.Done()
	return zbuf.NewArray(recs), nil
}
