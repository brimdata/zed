package proc

import (
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zson"
)

type Tail struct {
	Base
	limit int
	count int
	off   int
	q     []*zson.Record
}

func NewTail(c *Context, parent Proc, limit int) *Tail {
	q := make([]*zson.Record, limit)
	return &Tail{Base{Context: c, Parent: parent}, limit, 0, 0, q}
}

func (t *Tail) tail() zson.Batch {
	if t.count <= 0 {
		return nil
	}
	out := make([]*zson.Record, t.limit)
	for k := 0; k < t.limit; k++ {
		out[k] = t.q[(t.off+k)%t.limit]
	}
	t.off = 0
	t.count = 0
	return zson.NewArray(out, nano.NewSpanTs(t.MinTs, t.MaxTs))

}

func (t *Tail) Pull() (zson.Batch, error) {
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
