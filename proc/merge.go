package proc

import (
	"sync"

	"github.com/brimsec/zq/zbuf"
)

// A Merge proc merges multiple upstream inputs into one output.
//
// Note: rather than a standalone Proc, we could also have integrated
// merging behavior into proc.Base, which would simplify compilation a
// little bit. But we already took the standalone route for SplitProc,
// and this matches that.
type Merge struct {
	Base
	once     sync.Once
	ch       <-chan Result
	doneCh   chan struct{}
	parents  []*runnerProc
	nparents int
}

type runnerProc struct {
	Base
	ch     chan<- Result
	doneCh <-chan struct{}
}

func newrunnerProc(c *Context, parent Proc, ch chan<- Result, doneCh <-chan struct{}) *runnerProc {
	return &runnerProc{
		Base:   Base{Context: c, Parent: parent},
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

		r.ch <- Result{batch, err}
		if EOS(batch, err) {
			break
		}
	}
}

func NewMerge(c *Context, parents []Proc) *Merge {
	ch := make(chan Result)
	doneCh := make(chan struct{})
	var runners []*runnerProc
	for _, parent := range parents {
		runners = append(runners, newrunnerProc(c, parent, ch, doneCh))
	}
	p := Merge{
		Base:     Base{Context: c, Parent: nil},
		ch:       ch,
		doneCh:   doneCh,
		parents:  runners,
		nparents: len(parents),
	}
	return &p
}

func (m *Merge) Parents() []Proc {
	var pp []Proc
	for _, runner := range m.parents {
		pp = append(pp, runner.Parent)
	}
	return pp
}

// Pull implements the merge logic for returning data from the upstreams.
func (m *Merge) Pull() (zbuf.Batch, error) {
	m.once.Do(func() {
		for _, m := range m.parents {
			go m.run()
		}
	})

	for {
		res, ok := <-m.ch
		if !ok {
			return nil, nil
		}
		if res.Err != nil {
			m.Done()
			return nil, res.Err
		}

		if !EOS(res.Batch, res.Err) {
			return res.Batch, res.Err
		}

		m.nparents--
		if m.nparents == 0 {
			return nil, nil
		}
	}
}

func (m *Merge) Done() {
	close(m.doneCh)
}
