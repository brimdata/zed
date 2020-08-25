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
	once    sync.Once
	parents []mergeParent
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
	m := &Proc{ctx: c}
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
		if parent.batch == nil {
			select {
			case res := <-parent.resultCh:
				if res.Err != nil {
					return nil, res.Err
				}
				if res.Batch == nil {
					parent.done = true
					continue
				}
				parent.batch = res.Batch
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
		p.batch.Unref()
		p.batch = nil
	}
	return rec
}

func (m *Proc) Pull() (zbuf.Batch, error) {
	m.once.Do(m.run)
	return zbuf.ReadBatch(m, proc.BatchLen)
}

func (m *Proc) Done() {
	// XXX this should do something.  The new refactor of proc revealed
	// this problem because this proc embedded proc.Parent without using
	// it which has a nop done.  Everything is fine with a single flowgraph
	// path because this will eventually finish or get canceld but if
	// this is on a parallel paht with a downstream head proc, e.g.,
	// thtis Done should tear this proc down and signal Done on all of
	// the parents.  It doesn't look like this does this properly right now
	// and is instead relying on the context to finish for all the parents.
}
