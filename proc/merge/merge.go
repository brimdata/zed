package merge

import (
	"container/heap"
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
)

// Proc merges multiple upstream Pullers into one downstream Puller.
// If the input streams are ordered according to the configured comparison,
// the output of Merger will have the same order.  Each parent puller is run
// in its own goroutine so that deadlock is avoided when the upstream pullers
// would otherwise block waiting for an adjacent puller to finish but the
// Merger is waiting on the upstream puller.
type Proc struct {
	ctx  context.Context
	cmp  expr.CompareFn
	once sync.Once
	// parents holds all of the upstream pullers and never changes.
	parents []*puller
	// The head-of-line (hol) queue is maintained as a min-heap on cmp of
	// hol.vals[0] (see Less) so that the next Read always returns
	// hol[0].vals[0].
	hol     []*puller
	blocked map[*puller]struct{}
	err     error
}

var _ zbuf.Puller = (*Proc)(nil)
var _ zio.Reader = (*Proc)(nil)

func New(ctx context.Context, parents []proc.Interface, cmp expr.CompareFn) *Proc {
	pullers := make([]*puller, 0, len(parents))
	for _, p := range parents {
		pullers = append(pullers, newPuller(ctx, p))
	}
	return &Proc{
		ctx:     ctx,
		cmp:     cmp,
		parents: pullers,
		blocked: make(map[*puller]struct{}),
	}
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(p.run)
	if p.err != nil {
		return nil, p.err
	}
	if p.Len() == 0 {
		// No more batches in head of line.  So, lets resume
		// everything and return an EOS.
		for _, parent := range p.parents {
			parent.resumeCh <- struct{}{}
			ok, err := p.replenish(parent)
			if err != nil {
				p.err = err
				return nil, err
			}
			if ok {
				heap.Push(p, parent)
				delete(p.blocked, parent)
			}
		}
		return nil, nil
	}
	min := heap.Pop(p).(*puller)
	if p.Len() == 0 || p.cmp(&min.vals[len(min.vals)-1], &p.hol[0].vals[0]) <= 0 {
		// Either min is the only upstreams or min's last value is less
		// than or equal to the next upstream's first value.  Either
		// way, it's safe to return min's remaining values as a batch.
		batch := min.batch
		if len(min.vals) < len(batch.Values()) {
			batch = zbuf.NewArray(min.vals)
		}
		ok, err := p.replenish(min)
		if err != nil {
			p.err = err
			return nil, err
		}
		if ok {
			heap.Push(p, min)
		}
		return batch, nil
	}
	heap.Push(p, min)
	const batchLen = 100 // XXX
	vals := make([]zed.Value, 0, batchLen)
	for len(vals) < batchLen {
		val, err := p.Read()
		if err != nil {
			return nil, err
		}
		if val == nil {
			break
		}
		// Copy the underlying buffer because the next call to
		// zr.Read may overwrite it.
		vals = append(vals, *val.Copy())
	}
	if len(vals) == 0 {
		return nil, nil
	}
	return zbuf.NewArray(vals), nil
}

func (p *Proc) Read() (*zed.Value, error) {
	if p.Len() == 0 {
		return nil, nil
	}
	u := p.hol[0]
	val := &u.vals[0]
	u.vals = u.vals[1:]
	if len(u.vals) == 0 {
		ok, err := p.replenish(u)
		if err != nil {
			return nil, err
		}
		if !ok {
			heap.Pop(p)
		}
	}
	heap.Fix(p, 0)
	return val, nil
}

func (p *Proc) replenish(parent *puller) (bool, error) {
	ok, err := parent.replenish()
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
	}
	p.blocked[parent] = struct{}{}
	return false, nil
}

func (p *Proc) run() {
	// Start up all the goroutines before initializing the heap.
	// If we do one at a time, there is a deadlock for an upstream
	// split because the split waits for Pulls to arrive before
	// responding.
	for _, parent := range p.parents {
		go parent.run()
	}
	for _, parent := range p.parents {
		ok, err := p.replenish(parent)
		if err != nil {
			p.err = err
			return
		}
		if ok {
			p.Push(parent)
		}
	}
	heap.Init(p)
}

func (p *Proc) Done() {
	for _, parent := range p.hol {
		if _, ok := p.blocked[parent]; !ok {
			parent.doneCh <- struct{}{}
		}
	}
}

func (m *Proc) Len() int { return len(m.hol) }

func (m *Proc) Less(i, j int) bool {
	return m.cmp(&m.hol[i].vals[0], &m.hol[j].vals[0]) < 0
}

func (m *Proc) Swap(i, j int) { m.hol[i], m.hol[j] = m.hol[j], m.hol[i] }

func (m *Proc) Push(x interface{}) { m.hol = append(m.hol, x.(*puller)) }

func (m *Proc) Pop() interface{} {
	x := m.hol[len(m.hol)-1]
	m.hol = m.hol[:len(m.hol)-1]
	return x
}

type puller struct {
	proc.Interface
	ctx      context.Context
	resultCh chan proc.Result
	resumeCh chan struct{}
	doneCh   chan struct{}
	batch    zbuf.Batch
	vals     []zed.Value
}

func newPuller(ctx context.Context, parent proc.Interface) *puller {
	return &puller{
		Interface: proc.NewCatcher(parent),
		ctx:       ctx,
		resultCh:  make(chan proc.Result),
		resumeCh:  make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

func (p *puller) run() {
	defer close(p.resultCh)
	for {
		batch, err := p.Pull()
		select {
		case p.resultCh <- proc.Result{batch, err}:
			if err != nil {
				return
			}
			if batch == nil {
				if ok := p.resume(); !ok {
					return
				}
			}
		case <-p.doneCh:
			// Drop the pending batch and initiate a done...
			if ok := p.done(); !ok {
				return
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// replenish tries to receive the next batch.  It returns false when EOS
// is encountered and its goroutine will then block until resumed or
// canceled.
func (p *puller) replenish() (bool, error) {
	r := <-p.resultCh
	if r.Err != nil {
		return false, r.Err
	}
	p.batch = r.Batch
	if p.batch != nil {
		p.vals = p.batch.Values()
		return true, nil
	}
	return false, nil
}

func (p *puller) resume() bool {
	select {
	case <-p.resumeCh:
		return true
	case <-p.ctx.Done():
		return false
	}
}

func (p *puller) done() bool {
	p.Done()
	for {
		batch, err := p.Pull()
		if err != nil {
			p.resultCh <- proc.Result{nil, err}
			return false
		}
		if batch == nil {
			p.resultCh <- proc.Result{nil, nil}
			return p.resume()
		}
		batch.Unref()
	}
}
