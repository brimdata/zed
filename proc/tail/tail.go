package tail

import (
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Proc struct {
	parent proc.Interface
	limit  int
	count  int
	off    int
	q      []*zng.Record
}

func New(parent proc.Interface, limit int) *Proc {
	//XXX should have a limit check on limit
	return &Proc{
		parent: parent,
		limit:  limit,
		q:      make([]*zng.Record, limit),
	}
}

func (t *Proc) tail() zbuf.Batch {
	if t.count <= 0 {
		return nil
	}
	start := t.off
	if t.count < t.limit {
		start = 0
	}
	out := make([]*zng.Record, t.count)
	for k := 0; k < t.count; k++ {
		out[k] = t.q[(start+k)%t.limit]
	}
	t.off = 0
	t.count = 0
	return zbuf.NewArray(out)

}

func (t *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := t.parent.Pull()
		if proc.EOS(batch, err) {
			return t.tail(), nil
		}
		for k := 0; k < batch.Length(); k++ {
			t.q[t.off] = batch.Index(k).Keep()
			t.off = (t.off + 1) % t.limit
			t.count++
			if t.count >= t.limit {
				t.count = t.limit
			}
		}
		batch.Unref()
	}
}

func (p *Proc) Done() {
	p.parent.Done()
}
