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
	ctx     *proc.Context //XXX this parents method won't work; copy merge.Proc pattern?
	cmp     expr.CompareFn
	doneCh  chan struct{}
	once    sync.Once
	parents []mergeParent
}

type mergeParent struct {
	recs     []*zng.Record
	recIdx   int
	done     bool
	proc     proc.Interface
	resultCh chan proc.Result
}

func New(c *proc.Context, parents []proc.Interface, mergeField string, reversed bool) *Proc {
	m := &Proc{
		ctx:    c,
		doneCh: make(chan struct{}),
	}
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

func (p *Proc) run() {
	for i := range p.parents {
		parent := &p.parents[i]
		go func() {
			for {
				batch, err := parent.proc.Pull()
				select {
				case parent.resultCh <- proc.Result{batch, err}:
					if proc.EOS(batch, err) {
						return
					}
				case <-p.doneCh:
					parent.proc.Done()
					return
				case <-p.ctx.Done():
					return
				}
			}
		}()
	}
}

// Read fulfills zbuf.Reader so that we can use zbuf.ReadBatch.
func (p *Proc) Read() (*zng.Record, error) {
	idx := -1
	for i := range p.parents {
		parent := &p.parents[i]
		if parent.done {
			continue
		}
		if parent.recs == nil {
			select {
			case res := <-parent.resultCh:
				if res.Err != nil {
					return nil, res.Err
				}
				if res.Batch == nil {
					parent.done = true
					continue
				}
				// We're keeping records owned by res.Batch so don't call Unref.
				parent.recs = res.Batch.Records()
				parent.recIdx = 0
			case <-p.ctx.Done():
				return nil, p.ctx.Err()
			}
		}
		if idx == -1 || p.compare(parent, &p.parents[idx]) {
			idx = i
		}
	}
	if idx == -1 {
		return nil, nil
	}
	return next(&p.parents[idx]), nil
}

func (p *Proc) compare(x *mergeParent, y *mergeParent) bool {
	return p.cmp(x.recs[x.recIdx], y.recs[y.recIdx]) < 0
}

func next(p *mergeParent) *zng.Record {
	rec := p.recs[p.recIdx]
	p.recIdx++
	if p.recIdx == len(p.recs) {
		p.recs = nil
	}
	return rec
}

func (m *Proc) Pull() (zbuf.Batch, error) {
	m.once.Do(m.run)
	return zbuf.ReadBatch(m, proc.BatchLen)
}

func (m *Proc) Done() {
	close(m.doneCh)
}
