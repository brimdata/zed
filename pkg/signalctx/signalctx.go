// Package signalctx provides a context.Context that can be canceled by an
// operating system signal.
package signalctx

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

// New returns a context whose Done channel is closed when a provided signal (or
// any signal if none are provided) is received or when the returned cancel
// function is called, whichever happens first.
//
// Canceling this context releases resources associated with it, so code should
// call cancel as soon as the operations running in this context complete.
func New(sigs ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sctx := &signalCtx{Context: ctx}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	go func() {
		select {
		case sig := <-ch:
			sctx.setErr(fmt.Errorf("%s signal", sig))
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(ch)
	}()
	return sctx, func() {
		sctx.setErr(context.Canceled)
		cancel()
	}
}

type signalCtx struct {
	context.Context
	errMu sync.Mutex
	err   error
}

func (s *signalCtx) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

func (s *signalCtx) setErr(err error) {
	s.errMu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.errMu.Unlock()
}
