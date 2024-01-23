package tail

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
)

type Op struct {
	parent zbuf.Puller
	limit  int
	batch  zbuf.Batch
	count  int
	off    int
	q      []zed.Value
	eos    bool
}

func New(parent zbuf.Puller, limit int) *Op {
	//XXX should have a limit check on limit
	return &Op{
		parent: parent,
		limit:  limit,
		q:      make([]zed.Value, limit),
	}
}

func (o *Op) tail() zbuf.Batch {
	if o.count <= 0 {
		return nil
	}
	start := o.off
	if o.count < o.limit {
		start = 0
	}
	out := make([]zed.Value, o.count)
	for k := 0; k < o.count; k++ {
		out[k] = o.q[(start+k)%o.limit]
	}
	return zbuf.NewBatch(o.batch, out)

}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	if o.eos {
		// We don't check done here because if we already got EOS,
		// we don't propagate done.
		o.eos = false
		return nil, nil
	}
	if done {
		o.off = 0
		o.count = 0
		o.eos = false
		return o.parent.Pull(true)
	}
	for {
		batch, err := o.parent.Pull(false)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			batch = o.tail()
			if batch != nil {
				o.eos = true
				if o.batch != nil {
					o.batch.Unref()
					o.batch = nil
				}
			}
			o.off = 0
			o.count = 0
			return batch, nil
		}
		if o.batch == nil {
			batch.Ref()
			o.batch = batch
		}
		vals := batch.Values()
		for i := range vals {
			o.q[o.off] = vals[i].Copy()
			o.off = (o.off + 1) % o.limit
			o.count++
			if o.count >= o.limit {
				o.count = o.limit
			}
		}
		batch.Unref()
	}
}
