package tail

import (
	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng"
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

func (p *Proc) tail() zbuf.Batch {
	if p.count <= 0 {
		return nil
	}
	start := p.off
	if p.count < p.limit {
		start = 0
	}
	out := make([]*zng.Record, p.count)
	for k := 0; k < p.count; k++ {
		out[k] = p.q[(start+k)%p.limit]
	}
	p.off = 0
	p.count = 0
	return zbuf.Array(out)

}

func (p *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull()
		if proc.EOS(batch, err) {
			return p.tail(), nil
		}
		for k := 0; k < batch.Length(); k++ {
			p.q[p.off] = batch.Index(k).Keep()
			p.off = (p.off + 1) % p.limit
			p.count++
			if p.count >= p.limit {
				p.count = p.limit
			}
		}
		batch.Unref()
	}
}

func (p *Proc) Done() {
	p.parent.Done()
}
