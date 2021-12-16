package zbuf

import (
	"bytes"
	"container/heap"
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zio"
	"golang.org/x/sync/errgroup"
)

// A Merger merges multiple upstream Pullers into one downstream Puller.
// If the input streams are ordered according to the configured comparison,
// the output of Merger will have the same order.  Each parent puller is run
// in its own goroutine so that deadlock is avoided when the upstream pullers
// would otherwise block waiting for an adjacent puller to finish but the
// Merger is waiting on the upstream puller.
type Merger struct {
	cmp     expr.CompareFn
	pullers []Puller

	cancel context.CancelFunc
	ctx    context.Context
	group  *errgroup.Group
	once   sync.Once
	// Maintained as a min-heap on cmp of mergerUpstream.vals[0] (see Less)
	// so that the next Read always returns upstreams[0].vals[0].
	upstreams []*mergerUpstream
}

var _ Puller = (*Merger)(nil)
var _ zio.Reader = (*Merger)(nil)

func NewCompareFn(layout order.Layout) expr.CompareFn {
	nullsMax := layout.Order == order.Asc
	exprs := make([]expr.Evaluator, len(layout.Keys))
	for i, key := range layout.Keys {
		exprs[i] = expr.NewDottedExpr(key)
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
	group, ctx := errgroup.WithContext(ctx)
	return &Merger{
		cmp:     cmp,
		pullers: append([]Puller(nil), pullers...),
		cancel:  cancel,
		ctx:     ctx,
		group:   group,
	}
}

func (m *Merger) Cancel() {
	m.cancel()
}

func (m *Merger) Pull() (Batch, error) {
	m.once.Do(m.run)
	if m.Len() == 0 {
		return nil, m.group.Wait()
	}
	min := heap.Pop(m).(*mergerUpstream)
	if m.Len() == 0 || m.cmp(&min.vals[len(min.vals)-1], &m.upstreams[0].vals[0]) <= 0 {
		// Either min is the only upstreams or min's last value is less
		// than or equal to the next upstream's first value.  Either
		// way, it's safe to return min's remaining values as a batch.
		batch := min.batch
		if len(min.vals) < len(batch.Values()) {
			batch = NewArray(min.vals)
		}
		if min.receive() {
			heap.Push(m, min)
		}
		return batch, nil
	}
	heap.Push(m, min)
	const batchLen = 100 // XXX
	return readBatch(m, batchLen)
}

func (m *Merger) Read() (*zed.Value, error) {
	if m.Len() == 0 {
		return nil, m.group.Wait()
	}
	u := m.upstreams[0]
	zv := &u.vals[0]
	u.vals = u.vals[1:]
	if len(u.vals) > 0 || u.receive() {
		heap.Fix(m, 0)
	} else {
		heap.Pop(m)
	}
	return zv, nil
}

func (m *Merger) run() {
	for _, p := range m.pullers {
		p := p
		ch := make(chan Batch)
		m.group.Go(func() error {
			defer close(ch)
			for {
				batch, err := p.Pull()
				if batch == nil || err != nil {
					return err
				}
				select {
				case ch <- batch:
				case <-m.ctx.Done():
					return m.ctx.Err()
				}
			}
		})
		if u := (&mergerUpstream{ch: ch}); u.receive() {
			m.upstreams = append(m.upstreams, u)
		}
	}
	heap.Init(m)
}

func (m *Merger) Len() int { return len(m.upstreams) }

func (m *Merger) Less(i, j int) bool {
	return m.cmp(&m.upstreams[i].vals[0], &m.upstreams[j].vals[0]) < 0
}

func (m *Merger) Swap(i, j int) { m.upstreams[i], m.upstreams[j] = m.upstreams[j], m.upstreams[i] }

func (m *Merger) Push(x interface{}) { m.upstreams = append(m.upstreams, x.(*mergerUpstream)) }

func (m *Merger) Pop() interface{} {
	x := m.upstreams[len(m.upstreams)-1]
	m.upstreams = m.upstreams[:len(m.upstreams)-1]
	return x
}

type mergerUpstream struct {
	ch    <-chan Batch
	batch Batch
	vals  []zed.Value
}

// receive tries to receive the next batch.  It returns false when no batches
// remain.
func (m *mergerUpstream) receive() bool {
	batch, ok := <-m.ch
	m.batch = batch
	if m.batch != nil {
		m.vals = batch.Values()
	}
	return ok
}
