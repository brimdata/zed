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
	hol []*puller
}

var _ zbuf.Puller = (*Proc)(nil)
var _ zio.Reader = (*Proc)(nil)

func New(ctx context.Context, parents []zbuf.Puller, cmp expr.CompareFn) *Proc {
	pullers := make([]*puller, 0, len(parents))
	for _, p := range parents {
		pullers = append(pullers, newPuller(ctx, p))
	}
	return &Proc{
		ctx:     ctx,
		cmp:     cmp,
		parents: pullers,
	}
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	var err error
	p.once.Do(func() { err = p.run() })
	if err != nil {
		return nil, err
	}
	if p.Len() == 0 {
		// No more batches in head of line.  So, let's resume
		// everything and return an EOS.
		return nil, p.start()
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
		ok, err := min.replenish()
		if err != nil {
			return nil, err
		}
		if ok {
			heap.Push(p, min)
		}
		return batch, nil
	}
	heap.Push(p, min)
	const batchLen = 100 // XXX
	return zbuf.NewPuller(p, batchLen).Pull(false)
}

func (p *Proc) Read() (*zed.Value, error) {
	if p.Len() == 0 {
		return nil, nil
	}
	u := p.hol[0]
	val := &u.vals[0]
	u.vals = u.vals[1:]
	if len(u.vals) == 0 {
		ok, err := u.replenish()
		if err != nil {
			return nil, err
		}
		if !ok {
			heap.Pop(p)
			return val, nil
		}
	}
	heap.Fix(p, 0)
	return val, nil
}

func (p *Proc) run() error {
	// Start up all the goroutines before initializing the heap.
	// If we do one at a time, there is a deadlock for an upstream
	// split because the split waits for Pulls to arrive before
	// responding.
	for _, parent := range p.parents {
		go parent.run()
	}
	return p.start()
}

// start replenishes each parent's head-of-line batch either at initialization
// or after an EOS and intializes the blocked table with the status of
// each parent, e.g., a parent may be immediately blocked because it has
// no data at (re)start and should not be re-entered into the HOL queue.
func (p *Proc) start() error {
	p.hol = p.hol[:0]
	for _, parent := range p.parents {
		parent.blocked = false
		ok, err := parent.replenish()
		if err != nil {
			return err
		}
		if ok {
			heap.Push(p, parent)
		}
	}
	heap.Init(p)
	return nil
}

func (p *Proc) propagateDone() error {
	// For everything in the HOL queue (i.e., not already blocked),
	// propagate a done and read until EOS.  This will result in
	// all parents at EOS and blocked; then we can resume everything
	// together.
	for len(p.hol) > 0 {
		m := p.Pop().(*puller)
		select {
		case m.doneCh <- struct{}{}:
			if m.batch != nil {
				m.batch.Unref()
				m.batch = nil
			}
		case <-m.ctx.Done():
			return m.ctx.Err()
		}
	}
	// Now the heap is empty and all pullers are at EOS.
	// Unblock and initialize them so we can resume on the next bill.
	return p.start()
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
	zbuf.Puller
	ctx      context.Context
	resultCh chan proc.Result
	doneCh   chan struct{}
	batch    zbuf.Batch
	vals     []zed.Value
	// Used only by Proc
	blocked bool
}

func newPuller(ctx context.Context, parent zbuf.Puller) *puller {
	return &puller{
		Puller:   proc.NewCatcher(parent),
		ctx:      ctx,
		resultCh: make(chan proc.Result),
		doneCh:   make(chan struct{}),
	}
}

func (p *puller) run() {
	for {
		batch, err := p.Pull(false)
		select {
		case p.resultCh <- proc.Result{batch, err}:
			if err != nil {
				//XXX
				return
			}
		case <-p.doneCh:
			if batch != nil {
				batch.Unref()
			}
			// Drop the pending batch and initiate a done...
			batch, _ := p.Pull(true) // do something with err
			if batch != nil {
				panic("non-nil done batch")
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
	select {
	case r := <-p.resultCh:
		if r.Err != nil {
			return false, r.Err
		}
		p.batch = r.Batch
		if p.batch != nil {
			p.vals = p.batch.Values()
			return true, nil
		}
		p.blocked = true
		return false, nil
	case <-p.ctx.Done():
		return false, p.ctx.Err()
	}
}
