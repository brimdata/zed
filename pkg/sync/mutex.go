// +build !deadlock

package sync

import "sync"

type Mutex = sync.Mutex
type RWMutex = sync.RWMutex
