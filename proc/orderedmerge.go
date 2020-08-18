package proc

import (
	"sync"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// An OrderedMerge proc merges multiple upstream inputs into one output.
// If the input streams are ordered according to the configured comparison,
// the output of OrderedMerge will have the same order.
type OrderedMerge struct {
	Base
	recCmp  zbuf.RecordCmpFn
	once    sync.Once
	parents []mergeParent
}

type mergeParent struct {
	batch    zbuf.Batch
	recs     []*zng.Record
	recIdx   int
	done     bool
	proc     Proc
	resultCh chan Result
}

func NewOrderedMerge(c *Context, parents []Proc, cmp zbuf.RecordCmpFn) *OrderedMerge {
	m := &OrderedMerge{
		Base:    Base{Context: c},
		recCmp:  cmp,
		parents: make([]mergeParent, len(parents)),
	}
	for i := range parents {
		m.parents[i].proc = parents[i]
		m.parents[i].resultCh = make(chan Result)
	}
	return m
}

func (m *OrderedMerge) run() {
	for i := range m.parents {
		p := &m.parents[i]
		go func() {
			for {
				batch, err := p.proc.Pull()
				select {
				case p.resultCh <- Result{batch, err}:
					if EOS(batch, err) {
						return
					}
				case <-m.Context.Done():
					return
				}
			}
		}()
	}
}

// Read fulfills zbuf.Reader so that we can use zbuf.ReadBatch.
func (m *OrderedMerge) Read() (*zng.Record, error) {
	idx := -1
	for i := range m.parents {
		p := &m.parents[i]
		if p.done {
			continue
		}
		if p.batch == nil {
			select {
			case res := <-p.resultCh:
				if res.Err != nil {
					return nil, res.Err
				}
				if res.Batch == nil {
					p.done = true
					continue
				}
				p.batch = res.Batch
				p.recs = res.Batch.Records()
				p.recIdx = 0
			case <-m.Context.Done():
				return nil, m.Context.Err()
			}
		}
		if idx == -1 || m.compare(p, &m.parents[idx]) {
			idx = i
		}
	}
	if idx == -1 {
		return nil, nil
	}
	return m.next(&m.parents[idx]), nil
}

func (m *OrderedMerge) compare(x *mergeParent, y *mergeParent) bool {
	return m.recCmp(x.recs[x.recIdx], y.recs[y.recIdx])
}

func (m *OrderedMerge) next(p *mergeParent) *zng.Record {
	rec := p.recs[p.recIdx]
	p.recIdx++
	if p.recIdx == len(p.recs) {
		p.batch.Unref()
		p.batch = nil
	}
	return rec
}

func (m *OrderedMerge) Pull() (zbuf.Batch, error) {
	m.once.Do(m.run)
	return zbuf.ReadBatch(m, batchLen)
}
