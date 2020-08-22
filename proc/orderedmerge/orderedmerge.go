package orderedmerge

import (
	"sync"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// An OrderedMerge proc merges multiple upstream inputs into one output.
// If the input streams are ordered according to the configured comparison,
// the output of OrderedMerge will have the same order.
type Proc struct {
	proc.Parent //XXX this parents method won't work; copy merge.Proc pattern?
	cmp         expr.CompareFn
	once        sync.Once
	parents     []mergeParent
}

type mergeParent struct {
	batch    zbuf.Batch
	recs     []*zng.Record
	recIdx   int
	done     bool
	proc     proc.Interface
	resultCh chan proc.Result
}

func New(c *proc.Context, parents []proc.Interface, mergeField string, reversed bool) *Proc {
	m := &Proc{Parent: proc.Parent{Context: c}} //XXX
	cmpFn := expr.NewCompareFn(true, expr.CompileFieldAccess(mergeField))
	if !reversed {
		m.cmp = cmpFn
	} else {
		m.cmp = func(a, b *zng.Record) int { return cmpFn(b, a) }
	}
	m.parents = make([]mergeParent, len(parents))
	for i := range parents {
		m.parents[i].proc = parents[i]
		m.parents[i].resultCh = make(chan proc.Result)
	}
	return m
}

func (m *Proc) run() {
	for i := range m.parents {
		p := &m.parents[i]
		go func() {
			for {
				batch, err := p.proc.Pull()
				select {
				case p.resultCh <- proc.Result{batch, err}:
					if proc.EOS(batch, err) {
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
//XXX this is combining two things in one... I think this should be separated
// out and a proc can wrap the facter-out stuff
func (m *Proc) Read() (*zng.Record, error) {
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

func (m *Proc) compare(x *mergeParent, y *mergeParent) bool {
	return m.cmp(x.recs[x.recIdx], y.recs[y.recIdx]) < 0
}

func (m *Proc) next(p *mergeParent) *zng.Record {
	rec := p.recs[p.recIdx]
	p.recIdx++
	if p.recIdx == len(p.recs) {
		p.batch.Unref()
		p.batch = nil
	}
	return rec
}

const batchLen = 100 // XXX

func (m *Proc) Pull() (zbuf.Batch, error) {
	m.once.Do(m.run)
	return zbuf.ReadBatch(m, batchLen)
}
