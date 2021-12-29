package tail

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	parent zbuf.Puller
	limit  int
	count  int
	off    int
	q      []zed.Value
	eos    bool
}

func New(parent zbuf.Puller, limit int) *Proc {
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

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	if p.eos || done {
		p.eos = false
		return nil, nil
	}
	for {
		batch, err := p.parent.Pull(false)
		if batch == nil || err != nil {
			p.eos = true
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
