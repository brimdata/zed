package api

import (
	"context"
	"os"
	"os/signal"
)

type signalCtx struct {
	context.Context
	cancel  context.CancelFunc
	signals chan os.Signal
	caught  os.Signal
}

func newSignalCtx(sigs ...os.Signal) *signalCtx {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, sigs...)
	s := &signalCtx{signals: signals}
	s.Reset()
	return s
}

func (s *signalCtx) listen() {
	select {
	case s.caught = <-s.signals:
		s.cancel()
	case <-s.Done():
	}
}

func (s *signalCtx) Caught() os.Signal {
	<-s.Done()
	return s.caught
}

func (s *signalCtx) Reset() {
	if s.cancel != nil {
		s.cancel()
	}
	s.Context, s.cancel = context.WithCancel(context.Background())
	go s.listen()
}
