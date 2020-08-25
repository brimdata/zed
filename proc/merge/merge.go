package merge

import (
	"sync"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
)

// A Merge proc merges multiple upstream inputs into one output.
type Proc struct {
	once     sync.Once
	ch       <-chan proc.Result
	doneCh   chan struct{}
	parents  []*runnerProc
	nparents int
}

type runnerProc struct {
	parent proc.Interface
	ch     chan<- proc.Result
	doneCh <-chan struct{}
}

func newrunnerProc(parent proc.Interface, ch chan<- proc.Result, doneCh <-chan struct{}) *runnerProc {
	return &runnerProc{
		parent: parent,
		ch:     ch,
		doneCh: doneCh,
	}
}

func (r *runnerProc) run() {
	for {
		batch, err := r.parent.Pull()
		select {
		case _ = <-r.doneCh:
			r.parent.Done()
			break
		default:
		}

		r.ch <- proc.Result{batch, err}
		if proc.EOS(batch, err) {
			break
		}
	}
}

func New(c *proc.Context, parents []proc.Interface) *Proc {
	ch := make(chan proc.Result)
	doneCh := make(chan struct{})
	var runners []*runnerProc
	for _, parent := range parents {
		runners = append(runners, newrunnerProc(parent, ch, doneCh))
	}
	return &Proc{
		ch:       ch,
		doneCh:   doneCh,
		parents:  runners,
		nparents: len(parents),
	}
}

// Pull implements the merge logic for returning data from the upstreams.
func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() {
		for _, m := range p.parents {
			go m.run()
		}
	})
	for {
		if p.nparents == 0 {
			return nil, nil
		}
		res, ok := <-p.ch
		if !ok {
			return nil, nil
		}
		if res.Err != nil {
			p.Done()
			return nil, res.Err
		}

		if !proc.EOS(res.Batch, res.Err) {
			return res.Batch, res.Err
		}

		p.nparents--
	}
}

func (m *Proc) Done() {
	close(m.doneCh)
}
