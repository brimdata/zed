// +build deadlock

package sync

import (
	"time"

	deadlock "github.com/sasha-s/go-deadlock"
)

func init() {
	deadlock.Opts.OnPotentialDeadlock = func() {} // keep going when lock mis-ordering is encountered
	deadlock.Opts.DeadlockTimeout = 10 * time.Second
}

type Mutex = deadlock.Mutex
type RWMutex = deadlock.RWMutex
