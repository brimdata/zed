package proc

import (
	"io"
	"sync"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zq"
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
	parents []*runnerProc
	bufs    []zq.Batch
	err     error
}

type runnerProc struct {
	Base
	ch      chan Result
	proceed chan struct{}
}

func newrunnerProc(c *Context, parent Proc) *runnerProc {
	return &runnerProc{
		Base:    Base{Context: c, Parent: parent},
		ch:      make(chan Result),
		proceed: make(chan struct{}),
	}
}

func (r *runnerProc) run() {
	for {
		batch, err := r.Get()
		r.ch <- Result{batch, err}
		_, ok := <-r.proceed
		if !ok {
			// The downstream MergeProc closed us down.
			// Signal upstream that we're done and return
			// out of this goroutine.
			r.Done()
			return
		}
	}
}

func NewMerge(c *Context, parents []Proc) *Merge {
	var runners []*runnerProc
	for _, parent := range parents {
		runners = append(runners, newrunnerProc(c, parent))
	}
	p := Merge{Base: Base{Context: c, Parent: nil}, parents: runners}
	return &p
}

func (m *Merge) Parents() []Proc {
	var pp []Proc
	for _, runner := range m.parents {
		pp = append(pp, runner.Parent)
	}
	return pp
}

func (m *Merge) reload(k int) {
	parent := m.parents[k]
	result := <-parent.ch
	err := result.Err
	if err != nil && err != io.EOF {
		// If any parent has an error, we set this error,
		// which will cause the MergeProc to stop and return
		// the error, and run the done protocol on all the parents.
		m.err = err
	}
	buf := result.Batch
	m.bufs[k] = buf
	if buf == nil {
		close(parent.proceed)
		m.parents[k] = nil
	} else {
		parent.proceed <- struct{}{}
	}
}

// fill() initializes upstream data from each parent in our
// per-parent merge buffers.
func (m *Merge) fill() {
	for i := range m.parents {
		m.reload(i)
	}
}

// Pull implements the merge logic for returning data from the upstreams.
func (m *Merge) Pull() (zq.Batch, error) {
	m.once.Do(func() {
		for _, m := range m.parents {
			go m.run()
		}
		m.bufs = make([]zq.Batch, len(m.parents))
		m.fill()
	})
	if m.err != nil {
		m.Done()
		return nil, m.err
	}
	oldest := nano.MaxTs
	pick := -1

	// For now our "merge" just pushes out the batch with the oldest
	// timestamp at each round... this means that we may not
	// sending out monotonically ordered tiemstamps. Proper
	// time-ordered merging will come after Pull(span) is
	// implemented.
	for i, buf := range m.bufs {
		if buf == nil {
			continue
		}
		if buf.Span().Ts < oldest {
			oldest = buf.Span().Ts
			pick = i
		}
	}
	if pick == -1 {
		m.Done()
		return nil, nil
	}

	res := m.bufs[pick]
	m.reload(pick)
	if m.err != nil {
		m.Done()
		return nil, m.err
	}
	return res, nil
}

func (m *Merge) Done() {
	for k, parent := range m.parents {
		if parent != nil {
			<-parent.ch
			close(parent.proceed)
			m.parents[k] = nil
			m.bufs[k] = nil
		}
	}
}
