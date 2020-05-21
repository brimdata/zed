package cmd

import (
	"context"
	"os"
	"os/signal"
)

type signalCtx struct {
	context.Context
	sigs   []os.Signal
	caught os.Signal
}

func newSignalCtx(ctx context.Context, sigs ...os.Signal) (s *signalCtx) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, sigs...)
	childctx, cancel := context.WithCancel(ctx)
	s = &signalCtx{Context: childctx, sigs: sigs}
	go func() {
		select {
		case sig := <-signals:
			s.caught = sig
			cancel()
			return
		case <-childctx.Done():
			return
		}
	}()
	return
}

func (s *signalCtx) Caught() os.Signal {
	<-s.Done()
	return s.caught
}
