package proc

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
)

type labeled struct {
	zbuf.Batch
	label int
}

func Unwrap(batch zbuf.Batch) (zbuf.Batch, int) {
	var label int
	if inner, ok := batch.(*labeled); ok {
		batch = inner
		label = inner.label
	}
	return batch, label
}

// Mux implements the muxing of a set of parallel paths at the output of
// a flowgraph.  It also implements the double-EOS algorithm with proc.Latch
// to detect the end of each parallel stream.  Its output protocol is a single EOS
// when all of the upstream legs are done at which time it cancels the flowgraoh.
// Each  batch returned by the mux is wrapped in a Batch, which can be unwrappd
// with Unwrap to extract the integer index of the output (in left-to-right
// DFS traversal order of the flowgraph).  This proc requires more than one
// parent; use proc.Latcher for a single-output flowgraph.
type Mux struct {
	pctx     *Context
	once     sync.Once
	ch       <-chan result
	parents  []*puller
	nparents int
}

type result struct {
	batch zbuf.Batch
	label int
	err   error
}

type puller struct {
	Interface
	ch    chan<- result
	label int
}

func (p *puller) run(ctx context.Context) {
	for {
		batch, err := p.Pull()
		if err == EndOfBatch {
			continue
		}
		select {
		case p.ch <- result{batch, p.label, err}:
			if batch == nil || err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func NewMux(pctx *Context, parents []Interface) *Mux {
	if len(parents) <= 1 {
		panic("mux.New() must be called with two or more parents")
	}
	ch := make(chan result)
	pullers := make([]*puller, 0, len(parents))
	for label, parent := range parents {
		pullers = append(pullers, &puller{NewLatch(NewCatcher(parent)), ch, label})
	}
	return &Mux{
		pctx:     pctx,
		ch:       ch,
		parents:  pullers,
		nparents: len(parents),
	}
}

// Pull implements the merge logic for returning data from the upstreams.
func (m *Mux) Pull() (zbuf.Batch, error) {
	if m.nparents == 0 {
		// When we get to EOS, we make sure all the flowgraph
		// goroutines terminate by cacneling the proc context.
		m.pctx.Cancel()
		return nil, nil
	}
	m.once.Do(func() {
		for _, puller := range m.parents {
			go puller.run(m.pctx.Context)
		}
	})
	for {
		select {
		case res := <-m.ch:
			batch := res.batch
			err := res.err
			if err != nil {
				m.pctx.Cancel()
				return nil, err
			}
			if batch != nil {
				batch = &labeled{batch, res.label}
			} else {
				eoc := EndOfChannel(res.label)
				batch = &eoc
				m.nparents--
			}
			return batch, err
		case <-m.pctx.Context.Done():
			return nil, m.pctx.Context.Err()
		}
	}
}

func (m *Mux) Done() {
	panic("proc.Mux.Done() should not be called; instead proc.Context should be canceled.")
}

// EndOfChannel is an empty batch that represents the termination of one
// of the output paths of a muxed flowgraph and thus will be ignored downstream
// unless explicitly detected.
type EndOfChannel int

var _ zbuf.Batch = (*EndOfChannel)(nil)

func (*EndOfChannel) Ref()                  {}
func (*EndOfChannel) Unref()                {}
func (*EndOfChannel) Context() expr.Context { return nil }
func (*EndOfChannel) Values() []zed.Value   { return nil }
