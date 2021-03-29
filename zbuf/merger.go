package zbuf

import (
	"bytes"
	"context"
	"sync"

	"github.com/brimdata/zq/expr"
	"github.com/brimdata/zq/field"
	"github.com/brimdata/zq/zng"
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
}

type mergerPuller struct {
	Puller
	ch    chan batch
	recs  []*zng.Record
	batch Batch
}

type batch struct {
	Batch
	err error
}

func NewCompareFn(mergeField field.Static, reversed bool) expr.CompareFn {
	nullsMax := !reversed
	fn := expr.NewCompareFn(nullsMax, expr.NewDotExpr(mergeField))
	fn = totalOrderCompare(fn)
	if !reversed {
		return fn
	}
	return func(a, b *zng.Record) int { return fn(b, a) }
}

func totalOrderCompare(fn expr.CompareFn) expr.CompareFn {
	return func(a, b *zng.Record) int {
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

func MergeReadersByTsAsReader(ctx context.Context, readers []Reader, order Order) (Reader, error) {
	if len(readers) == 1 {
		return readers[0], nil
	}
	return MergeReadersByTs(ctx, readers, order)
}

func MergeReadersByTs(ctx context.Context, readers []Reader, order Order) (*Merger, error) {
	pullers, err := ReadersToPullers(ctx, readers)
	if err != nil {
		return nil, err
	}
	return MergeByTs(ctx, pullers, order), nil
}

func MergeByTs(ctx context.Context, pullers []Puller, order Order) *Merger {
	cmp := func(a, b *zng.Record) int {
		if order == OrderDesc {
			a, b = b, a
		}
		aTs, bTs := a.Ts(), b.Ts()
		if aTs < bTs {
			return -1
		}
		if aTs > bTs {
			return 1
		}
		return bytes.Compare(a.Bytes, b.Bytes)
	}
	return NewMerger(ctx, pullers, cmp)
}

func (m *Merger) run() {
	for _, p := range m.pullers {
		puller := p
		go func() {
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
func (m *Merger) Read() (*zng.Record, error) {
	m.once.Do(m.run)
	leader, err := m.findLeader()
	if leader < 0 || err != nil {
		m.Cancel()
		return nil, err
	}
	return m.pullers[leader].next(), nil
}

func (m *mergerPuller) next() *zng.Record {
	rec := m.recs[0]
	m.recs = m.recs[1:]
	m.batch = nil
	return rec
}

func (m *Merger) findLeader() (int, error) {
	leader := -1
	for k, p := range m.pullers {
		if p == nil {
			continue
		}
		if len(p.recs) == 0 {
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
				p.recs = b.Batch.Records()
				p.batch = b.Batch
			case <-m.ctx.Done():
				return -1, m.ctx.Err()
			}
		}
		if leader == -1 || m.cmp(p.recs[0], m.pullers[leader].recs[0]) < 0 {
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
	last := hol.recs[len(hol.recs)-1]
	for k, p := range m.pullers {
		if k == leader || p == nil {
			continue
		}
		if m.cmp(last, p.recs[0]) > 0 {
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
		return nil, err
	}
	if !m.overlaps(leader) {
		b := m.pullers[leader].batch
		m.pullers[leader].recs = nil
		return b, nil
	}
	const batchLen = 100 // XXX
	return readBatch(m, batchLen)
}

func (m *Merger) Cancel() {
	m.cancel()
}
