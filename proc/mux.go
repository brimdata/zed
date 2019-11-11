package proc

import (
	"errors"
	"sync"
	"time"
)

type MuxResult struct {
	Result
	ID      int
	Warning string
}

type MuxOutput struct {
	ctx      *Context
	runners  int
	muxProcs []*MuxProc
	once     sync.Once
	in       chan MuxResult
}

type MuxProc struct {
	Base
	ID  int
	out chan<- MuxResult
}

func newMuxProc(c *Context, parent Proc, id int, out chan MuxResult) *MuxProc {
	return &MuxProc{Base: Base{Context: c, Parent: parent}, ID: id, out: out}
}

func (m *MuxProc) run() {
	// This loop pulls batches from the parent and pushes them
	// downstream to the multiplexing proc.  If the mux isn't ready,
	// the out channel will block and this  goroutine will block until
	// that downstream path becomes ready.  This, in turn, causes the
	// mux to run at the rate of the ultimate output path so that
	// we are flow-controlled here and do not build up large queues
	// due to rate mismatch.
	for {
		batch, err := m.Get()
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
		mux.muxProcs = append(mux.muxProcs, newMuxProc(ctx, parent, id, c))
	}
	return mux
}

func (m *MuxOutput) Complete() bool {
	return m.runners <= 0
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
