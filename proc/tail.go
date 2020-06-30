package proc

import (
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Tail struct {
	Base
	limit int
	count int
	off   int
	q     []*zng.Record
}

func NewTail(c *Context, parent Proc, limit int) *Tail {
	q := make([]*zng.Record, limit)
	return &Tail{Base{Context: c, Parent: parent}, limit, 0, 0, q}
}

func (t *Tail) tail() zbuf.Batch {
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

func (t *Tail) Pull() (zbuf.Batch, error) {
	for {
		batch, err := t.Get()
		if EOS(batch, err) {
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
