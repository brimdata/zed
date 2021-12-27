package proc

import (
	"errors"

	"github.com/brimdata/zed/zbuf"
)

// EndOfBatch is returned by Latch.Pull to signal the end of a batch.
// It should be handled by a caller of Latch.Pull() and not propagated downstream.
var EndOfBatch = errors.New("end of batch")

// Latch is an operator that converts the double EOS protocol into a
// the single EOS protocol. A caller of Latch.Pull() can presume
// EOS really means EOS but cannot detect batch boundaries.
type Latch struct {
	parent Interface
	eos    bool
	done   bool
	err    error
}

var _ Interface = (*Latch)(nil)

func NewLatch(parent Interface) *Latch {
	return &Latch{parent: parent}
}

func (l *Latch) Pull() (zbuf.Batch, error) {
	if l.done {
		return nil, l.err
	}
	b, err := l.parent.Pull()
	if err != nil {
		if err == EndOfBatch {
			// This breaks the protocol and shouldn't happen so we panic.
			panic("proc.Latch received EndOfBatch")
		}
		l.err = err
		l.done = true
		return nil, err
	}
	if b == nil {
		if l.eos {
			l.done = true
			return nil, nil
		}
		l.eos = true
		return nil, EndOfBatch
	}
	l.eos = false
	return b, nil
}

func (l *Latch) Done() {
	l.parent.Done()
	l.done = true
}

// Latcher is used for a single output flowgraph.  It translates the double EOS
// protocol into a single EOS and cancels the proc context at EOS.
type Latcher struct {
	pctx   *Context
	parent *Latch
	err    error
	eos    bool
}

func NewLatcher(pctx *Context, parent Interface) *Latcher {
	return &Latcher{
		pctx:   pctx,
		parent: NewLatch(parent),
	}
}

func (l *Latcher) Pull() (zbuf.Batch, error) {
	if l.eos {
		l.pctx.Cancel()
		return nil, nil
	}
	for {
		batch, err := l.parent.Pull()
		if err != nil {
			if err == EndOfBatch {
				continue
			}
			l.eos = true
			l.pctx.Cancel()
			return nil, err
		}
		if batch == nil {
			l.eos = true
			eoc := EndOfChannel(0)
			return &eoc, nil
		}
		return batch, err
	}
}

func (l *Latcher) Done() {
	panic("proc.Latcher.Done() should not be called; instead proc.Context should be canceled.")
}
