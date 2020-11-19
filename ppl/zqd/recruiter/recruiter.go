package recruiter

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// WorkerPool is the internal state of the recruiter system
// The pools are exported only for use in unit tests
type WorkerPool struct {
	mu           sync.Mutex                // one lock for all three maps
	freePool     map[string]WorkerDetail   // Map of all free workers
	nodePool     map[string][]WorkerDetail // Map of nodes of slices of free workers
	reservedPool map[string]WorkerDetail   // Map of busy workers
	SkipSpread   bool                      // option to test algorithm performance
	r            *rand.Rand
}

type WorkerDetail struct {
	Addr     string
	NodeName string
}

func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		freePool:     make(map[string]WorkerDetail),
		nodePool:     make(map[string][]WorkerDetail),
		reservedPool: make(map[string]WorkerDetail),
		r:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (pool *WorkerPool) Register(addr string, nodename string) error {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Errorf("invalid address for Register: %w", err)
	}
	if nodename == "" {
		return fmt.Errorf("node name required for Register")
	}
	wd := WorkerDetail{Addr: addr, NodeName: nodename}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	if _, ok := pool.reservedPool[addr]; ok {
		return nil // ignore register for existing workers
	}
	pool.freePool[addr] = wd
	pool.nodePool[nodename] = append(pool.nodePool[nodename], wd)
	return nil
}

// removeFromNodePool is internal and the calling function must be holding the mutex
func (pool *WorkerPool) removeFromNodePool(wd WorkerDetail) {
	s := pool.nodePool[wd.NodeName]
	if len(s) == 1 {
		if s[0] == wd {
			// Remove empty list from the hash so len(pool.nodePool)
			// is a a count of nodes with available workers.
			delete(pool.nodePool, wd.NodeName)
		}
		return
	}
	i := -1
	for j, v := range s {
		if v == wd {
			i = j
			break
		}
	}
	if i == -1 {
		panic(fmt.Errorf("expected WorkerDetail not in list: %v", wd.Addr))
	}
	// Overwrite removed node and truncate slice
	s[i] = s[len(s)-1]
	pool.nodePool[wd.NodeName] = s[:len(s)-1]
}

func (pool *WorkerPool) Deregister(addr string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if wd, ok := pool.freePool[addr]; ok {
		pool.removeFromNodePool(wd)
		delete(pool.freePool, addr)
	}
}

// Unreserve removes from the reserved pool, but does not reregister.
// The worker process should initiate register.
func (pool *WorkerPool) Unreserve(addr string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	delete(pool.reservedPool, addr)
}

// Recruit attempts to spread the workers across nodes.
func (pool *WorkerPool) Recruit(n int) ([]WorkerDetail, error) {
	if n < 1 {
		return nil, fmt.Errorf("recruit must request one or more workers: n=%d", n)
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	var recruits []WorkerDetail
	if len(pool.nodePool) < 1 {
		return recruits, nil
	}

	// Make a single pass through the nodes in the cluster that have
	// available workers, and try to pick evenly from each node. If that pass
	// fails to recruit enough workers, then we start to pick from the freePool, regardless
	// of node, until we recruit enough workers, or all available workers.
	if !pool.SkipSpread && n > 1 {
		var keys []string
		for k, _ := range pool.nodePool {
			keys = append(keys, k)
		}
		pool.r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
		for i, key := range keys {
			workers := pool.nodePool[key]
			// adjust goal on each iteration
			d := len(keys) - i
			goal := (n - len(recruits) + d - 1) / d
			if len(workers) > goal {
				recruits = append(recruits, workers[:goal]...)
				pool.nodePool[key] = workers[goal:]
			} else {
				recruits = append(recruits, workers...)
				delete(pool.nodePool, key)
			}
			if len(recruits) == n {
				break
			}
		}

		// Delete the recruits obtained in this pass from the freePool
		for _, wd := range recruits {
			_, prs := pool.freePool[wd.Addr]
			if !prs {
				panic(fmt.Errorf("attempt to remove addr that was not in freePool: %v", wd.Addr))
			}
			delete(pool.freePool, wd.Addr)
		}
	}

	// If there are still recruits needed, select them by iterating through the freePool
	if len(recruits) < n {
		for k, wd := range pool.freePool {
			pool.removeFromNodePool(wd)
			delete(pool.freePool, k)
			recruits = append(recruits, wd)
			if len(recruits) == n {
				break
			}
		}
	}

	// Add the recruits to the Reserved Pool
	for _, r := range recruits {
		pool.reservedPool[r.Addr] = r
	}
	return recruits, nil
}

func (pool *WorkerPool) LenFreePool() int {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return len(pool.freePool)
}

func (pool *WorkerPool) LenReservedPool() int {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return len(pool.reservedPool)
}

func (pool *WorkerPool) LenNodePool() int {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	return len(pool.nodePool)
}
