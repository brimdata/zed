package exprswitch

import (
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type ExprSwitch struct {
	parent    proc.Interface
	evaluator expr.Evaluator

	cases     map[string]chan<- *zed.Value
	defaultCh chan<- *zed.Value
	doneChCh  chan chan<- *zed.Value
	err       error
	once      sync.Once
}

func New(parent proc.Interface, e expr.Evaluator) *ExprSwitch {
	return &ExprSwitch{
		parent:    parent,
		evaluator: e,
		cases:     make(map[string]chan<- *zed.Value),
		doneChCh:  make(chan chan<- *zed.Value),
	}
}

func (s *ExprSwitch) NewProc(zv zed.Value) proc.Interface {
	ch := make(chan *zed.Value)
	if zv.IsNil() {
		s.defaultCh = ch
	} else {
		s.cases[string(zv.Bytes)] = ch
	}
	return &Proc{s, ch}
}

func (s *ExprSwitch) run() {
	defer func() {
		for _, ch := range s.cases {
			close(ch)
		}
		if s.defaultCh != nil {
			close(s.defaultCh)
		}
		s.parent.Done()
	}()
	for {
		batch, err := s.parent.Pull()
		if proc.EOS(batch, err) {
			s.err = err
			return
		}
		scope := batch.Scope()
		vals := batch.Values()
		for i := range vals {
			val := s.evaluator.Eval(&vals[i], scope)
			if val == zed.Missing {
				continue
			}
		again:
			ch, ok := s.cases[string(val.Bytes)]
			if !ok {
				ch = s.defaultCh
			}
			if ch == nil {
				continue
			}
			select {
			case ch <- &vals[i]:
			case doneCh := <-s.doneChCh:
				s.handleDoneCh(doneCh)
				if len(s.cases) == 0 && s.defaultCh == nil {
					return
				}
				goto again
			}
		}
	}
}

func (s *ExprSwitch) handleDoneCh(doneCh chan<- *zed.Value) {
	if s.defaultCh == doneCh {
		s.defaultCh = nil
	} else {
		for k, ch := range s.cases {
			if ch == doneCh {
				delete(s.cases, k)
				break
			}
		}
	}
}

type Proc struct {
	parent *ExprSwitch
	ch     <-chan *zed.Value
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.parent.once.Do(func() {
		go p.parent.run()
	})
	if val, ok := <-p.ch; ok {
		//XXX we should make this more efficient by pushing batches
		// instead of values over the channel like split does.
		return zbuf.NewArray([]zed.Value{*val}), nil
	}
	return nil, p.parent.err
}

func (p *Proc) Done() {
	go func() {
		for {
			if _, ok := <-p.ch; !ok {
				return
			}
		}
	}()
}
