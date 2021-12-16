package tail

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	parent proc.Interface
	limit  int
	count  int
	off    int
	q      []zed.Value
}

func New(parent proc.Interface, limit int) *Proc {
	//XXX should have a limit check on limit
	return &Proc{
		parent: parent,
		limit:  limit,
		q:      make([]zed.Value, limit),
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
	out := make([]zed.Value, p.count)
	for k := 0; k < p.count; k++ {
		out[k] = p.q[(start+k)%p.limit]
	}
	p.off = 0
	p.count = 0
	return zbuf.NewArray(out)

}

func (p *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull()
		if proc.EOS(batch, err) {
			return p.tail(), nil
		}
		vals := batch.Values()
		for i := range vals {
			p.q[p.off] = *vals[i].Copy()
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
