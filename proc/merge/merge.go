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
	proc.Parent
	ch     chan<- proc.Result
	doneCh <-chan struct{}
}

func newrunnerProc(parent proc.Parent, ch chan<- proc.Result, doneCh <-chan struct{}) *runnerProc {
	return &runnerProc{
		Parent: parent,
		ch:     ch,
		doneCh: doneCh,
	}
}

func (r *runnerProc) run() {
	for {
		batch, err := r.Get()
		select {
		case _ = <-r.doneCh:
			r.Parent.Done()
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
		runners = append(runners, newrunnerProc(c.NewParent(parent), ch, doneCh))
	}
	return &Proc{
		ch:       ch,
		doneCh:   doneCh,
		parents:  runners,
		nparents: len(parents),
	}
}

func (m *Proc) Parents() []proc.Interface {
	var pp []proc.Interface
	for _, runner := range m.parents {
		pp = append(pp, runner.Parent.Interface)
	}
	return pp
}

// Pull implements the merge logic for returning data from the upstreams.
func (m *Proc) Pull() (zbuf.Batch, error) {
	m.once.Do(func() {
		for _, m := range m.parents {
			go m.run()
		}
	})
	for {
		if m.nparents == 0 {
			return nil, nil
		}
		res, ok := <-m.ch
		if !ok {
			return nil, nil
		}
		if res.Err != nil {
			m.Done()
			return nil, res.Err
		}

		if !proc.EOS(res.Batch, res.Err) {
			return res.Batch, res.Err
		}

		m.nparents--
	}
}

func (m *Proc) Done() {
	close(m.doneCh)
}
