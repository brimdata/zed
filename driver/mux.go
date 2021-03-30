package driver

import (
	"errors"
	"sync"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zqe"
)

type muxResult struct {
	proc.Result
	ID      int
	Warning string
}

type muxOutput struct {
	pctx     *proc.Context
	runners  int
	muxProcs []*mux
	once     sync.Once
	in       chan muxResult
	scanner  zbuf.Statser
}

type mux struct {
	parent proc.Interface
	ID     int
	out    chan<- muxResult
}

func newMux(parent proc.Interface, id int, out chan muxResult) *mux {
	return &mux{
		parent: parent,
		ID:     id,
		out:    out,
	}
}

func (m *mux) safeGet() (b zbuf.Batch, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err = zqe.RecoverError(r)
	}()
	b, err = m.parent.Pull()
	return
}

func (m *mux) run() {
	// This loop pulls batches from the parent and pushes them
	// downstream to the multiplexing proc.  If the mux isn't ready,
	// the out channel will block and this  goroutine will block until
	// that downstream path becomes ready.  This, in turn, causes the
	// mux to run at the rate of the ultimate output path so that
	// we are flow-controlled here and do not build up large queues
	// due to rate mismatch.
	for {
		batch, err := m.safeGet()
		m.out <- muxResult{proc.Result{batch, err}, m.ID, ""}
		if proc.EOS(batch, err) {
			return
		}
	}
}

func newMuxOutput(pctx *proc.Context, parents []proc.Interface, scanner zbuf.Statser) *muxOutput {
	n := len(parents)
	c := make(chan muxResult, n)
	mux := &muxOutput{pctx: pctx, runners: n, in: c, scanner: scanner}
	for id, parent := range parents {
		mux.muxProcs = append(mux.muxProcs, newMux(parent, id, c))
	}
	return mux
}

func (m *muxOutput) Stats() api.ScannerStats {
	if m.scanner == nil {
		return api.ScannerStats{}
	}
	return api.ScannerStats(*m.scanner.Stats())
}

func (m *muxOutput) Complete() bool {
	return len(m.pctx.Warnings) == 0 && m.runners == 0
}

var errTimeout = errors.New("timeout")

func (m *muxOutput) Pull(timeout <-chan time.Time) muxResult {
	m.once.Do(func() {
		for _, m := range m.muxProcs {
			go m.run()
		}
	})
	if m.Complete() {
		return muxResult{proc.Result{}, 0, ""}
	}
	var result muxResult
	select {
	case <-timeout:
		return muxResult{proc.Result{nil, errTimeout}, 0, ""}
	case result = <-m.in:
		// empty
	case warning := <-m.pctx.Warnings:
		return muxResult{proc.Result{}, 0, warning}
	}

	if proc.EOS(result.Batch, result.Err) {
		m.runners--
	}
	return result
}

func (m *muxOutput) Drain() {
	for !m.Complete() {
		m.Pull(nil)
	}
}
