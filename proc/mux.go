package proc

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/brimsec/zq/zbuf"
)

type MuxResult struct {
	Result
	ID      int
	Warning string
}

type MuxOutput struct {
	ctx      *Context
	runners  int
	muxProcs []*Mux
	once     sync.Once
	in       chan MuxResult
}

type Mux struct {
	Base
	ID  int
	out chan<- MuxResult
}

func newMux(c *Context, parent Proc, id int, out chan MuxResult) *Mux {
	return &Mux{Base: Base{Context: c, Parent: parent}, ID: id, out: out}
}

func (m *Mux) safeGet() (b zbuf.Batch, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err = fmt.Errorf("panic: %+v", r)
	}()
	b, err = m.Get()
	return
}

func (m *Mux) run() {
	// This loop pulls batches from the parent and pushes them
	// downstream to the multiplexing proc.  If the mux isn't ready,
	// the out channel will block and this  goroutine will block until
	// that downstream path becomes ready.  This, in turn, causes the
	// mux to run at the rate of the ultimate output path so that
	// we are flow-controlled here and do not build up large queues
	// due to rate mismatch.
	for {
		batch, err := m.safeGet()
		m.out <- MuxResult{Result{batch, err}, m.ID, ""}
		if EOS(batch, err) {
			return
		}
	}
}

func NewMuxOutput(ctx *Context, parents []Proc) *MuxOutput {
	n := len(parents)
	c := make(chan MuxResult, n)
	mux := &MuxOutput{ctx: ctx, runners: n, in: c}
	for id, parent := range parents {
		mux.muxProcs = append(mux.muxProcs, newMux(ctx, parent, id, c))
	}
	return mux
}

func (m *MuxOutput) Complete() bool {
	return m.runners <= 0
}

func (m *MuxOutput) N() int {
	return len(m.muxProcs)
}

//XXX
var ErrTimeout = errors.New("timeout")

func (m *MuxOutput) Pull(timeout <-chan time.Time) MuxResult {
	m.once.Do(func() {
		for _, m := range m.muxProcs {
			go m.run()
		}
	})
	if m.Complete() {
		return MuxResult{Result{}, -1, ""}
	}
	var result MuxResult
	if timeout == nil {
		result = <-m.in
	} else {
		select {
		case <-timeout:
			return MuxResult{Result{nil, ErrTimeout}, 0, ""}
		case result = <-m.in:
			// empty
		case warning := <-m.ctx.Warnings:
			return MuxResult{Result{}, 0, warning}
		}
	}
	if EOS(result.Batch, result.Err) {
		m.runners--
	}
	return result
}

func (m *MuxOutput) Drain() {
	for !m.Complete() {
		m.Pull(nil)
	}
}
