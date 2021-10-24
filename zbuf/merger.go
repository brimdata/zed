package zbuf

import (
	"bytes"
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/order"
)

// A Merger merges multiple upstream Pullers into one downstream Puller.
// If the input streams are ordered according to the configured comparison,
// the output of Merger will have the same order.  Each parent puller is run
// in its own goroutine so that deadlock is avoided when the upstream pullers
// would otherwise block waiting for an adjacent puller to finish but the
// Merger is waiting on the upstream puller.
type Merger struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cmp     expr.CompareFn
	once    sync.Once
	pullers []*mergerPuller
	wg      sync.WaitGroup
}

type mergerPuller struct {
	Puller
	ch    chan batch
	batch Batch
	zvals []zed.Value
}

type batch struct {
	Batch
	err error
}

func NewCompareFn(layout order.Layout) expr.CompareFn {
	nullsMax := layout.Order == order.Asc
	exprs := make([]expr.Evaluator, len(layout.Keys))
	for i, key := range layout.Keys {
		exprs[i] = expr.NewDotExpr(key)
	}
	fn := expr.NewCompareFn(nullsMax, exprs...)
	fn = totalOrderCompare(fn)
	if layout.Order == order.Asc {
		return fn
	}
	return func(a, b *zed.Value) int { return fn(b, a) }
}

func totalOrderCompare(fn expr.CompareFn) expr.CompareFn {
	return func(a, b *zed.Value) int {
		cmp := fn(a, b)
		if cmp == 0 {
			return bytes.Compare(a.Bytes, b.Bytes)
		}
		return cmp
	}
}

func NewMerger(ctx context.Context, pullers []Puller, cmp expr.CompareFn) *Merger {
	ctx, cancel := context.WithCancel(ctx)
	m := &Merger{
		ctx:    ctx,
		cmp:    cmp,
		cancel: cancel,
	}
	m.pullers = make([]*mergerPuller, 0, len(pullers))
	for _, p := range pullers {
		m.pullers = append(m.pullers, &mergerPuller{Puller: p, ch: make(chan batch)})
	}
	return m
}

func (m *Merger) run() {
	for _, p := range m.pullers {
		m.wg.Add(1)
		puller := p
		go func() {
			defer m.wg.Done()
			for {
				b, err := puller.Pull()
				select {
				case puller.ch <- batch{Batch: b, err: err}:
					if b == nil && err == nil {
						// EOS
						return
					}
				case <-m.ctx.Done():
					return
				}
			}
		}()
	}
}

// Read fulfills Reader so that we can use ReadBatch or
// use Merger as a Reader directly.
func (m *Merger) Read() (*zed.Value, error) {
	m.once.Do(m.run)
	leader, err := m.findLeader()
	if leader < 0 || err != nil {
		m.Cancel()
		m.wg.Wait()
		return nil, err
	}
	return m.pullers[leader].next(), nil
}

func (m *mergerPuller) next() *zed.Value {
	rec := m.zvals[0]
	m.zvals = m.zvals[1:]
	m.batch = nil
	return &rec
}

func (m *Merger) findLeader() (int, error) {
	leader := -1
	for k, p := range m.pullers {
		if p == nil {
			continue
		}
		if len(p.zvals) == 0 {
			select {
			case b := <-p.ch:
				if b.err != nil {
					return -1, b.err
				}
				if b.Batch == nil {
					// EOS
					m.pullers[k] = nil
					continue
				}
				// We're keeping records owned by res.Batch so don't call Unref.
				// XXX this means the batch won't be returned to
				// the pool and instead will run through GC.
				p.zvals = b.Batch.Values()
				p.batch = b.Batch
			case <-m.ctx.Done():
				return -1, m.ctx.Err()
			}
		}
		if leader == -1 || m.cmp(&p.zvals[0], &m.pullers[leader].zvals[0]) < 0 {
			leader = k
		}
	}
	return leader, nil
}

func (m *Merger) overlaps(leader int) bool {
	hol := m.pullers[leader]
	if hol.batch == nil {
		return true
	}
	last := &hol.zvals[len(hol.zvals)-1]
	for k, p := range m.pullers {
		if k == leader || p == nil {
			continue
		}
		if m.cmp(last, &p.zvals[0]) > 0 {
			return true
		}
	}
	return false
}

func (m *Merger) Pull() (Batch, error) {
	m.once.Do(m.run)
	leader, err := m.findLeader()
	if leader < 0 || err != nil {
		m.Cancel()
		m.wg.Wait()
		return nil, err
	}
	if !m.overlaps(leader) {
		b := m.pullers[leader].batch
		m.pullers[leader].zvals = nil
		return b, nil
	}
	const batchLen = 100 // XXX
	return readBatch(m, batchLen)
}

func (m *Merger) Cancel() {
	m.cancel()
}
