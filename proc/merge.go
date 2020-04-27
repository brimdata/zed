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
	once    sync.Once
	ch      <-chan Result
	parents []*runnerProc
}

type runnerProc struct {
	Base
	ch chan<- Result
}

func newrunnerProc(c *Context, parent Proc, ch chan<- Result) *runnerProc {
	return &runnerProc{
		Base: Base{Context: c, Parent: parent},
		ch:   ch,
	}
}

func (r *runnerProc) run() {
	for {
		batch, err := r.Get()
		r.ch <- Result{batch, err}
		if EOS(batch, err) {
			break
		}
	}
}

func NewMerge(c *Context, parents []Proc) *Merge {
	ch := make(chan Result)
	var runners []*runnerProc
	for _, parent := range parents {
		runners = append(runners, newrunnerProc(c, parent, ch))
	}
	p := Merge{
		Base:    Base{Context: c, Parent: nil},
		ch:      ch,
		parents: runners,
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

	res, ok := <-m.ch
	if !ok {
		return nil, nil
	}
	return res.Batch, res.Err
}
