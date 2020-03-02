// package catcher is a convient wrapper to interconnect signals with
// contexts, etc
package catcher

import (
	"context"
	"os"
	"os/signal"
)

type Catcher interface {
	Catch(ctx context.Context) (context.Context, context.CancelFunc)
}

type SignalCatcher struct {
	sigs []os.Signal
}

func NewSignalCatcher(sigs ...os.Signal) *SignalCatcher {
	return &SignalCatcher{sigs: sigs}
}

func (s *SignalCatcher) Catch(ctx context.Context) (context.Context, context.CancelFunc) {
	//XXX this could be made more efficient by having the signal handler
	// be a parent of each context created and keeping just one signal
	// channel around.  then when we get the signal, we cancel the parent
	// context and all the children get canceled.
	// but for now, this is ok
	signals := make(chan os.Signal, 1)
	for _, sig := range s.sigs {
		signal.Notify(signals, sig)
	}
	childctx, cancel := context.WithCancel(ctx)
	go func() {
		// The for loop waits until we get a signal or a done.
		// When we're done, we return and this goroutine exits.
		// When we get a signal, we call cancel and continue the
		// loop and wait for the done to finally make us leave.
		for {
			select {
			case <-signals:
				cancel()
			case <-childctx.Done():
				return
			}
		}
	}()
	return childctx, cancel
}
