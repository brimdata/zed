package recruiter

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"
)

// WorkerPool is the internal state of the recruiter system
// The pools are exported only for use in unit tests
type WorkerPool struct {
	mu           sync.Mutex                // for right now, just share one lock for all three maps
	freePool     map[string]WorkerDetail   // Map of all free workers
	nodePool     map[string][]WorkerDetail // Map of nodes of slices of free workers
	reservedPool map[string]WorkerDetail   // Map of busy workers
}

type WorkerDetail struct {
	Addr     string
	NodeName string
}

func NewWorkerPool() *WorkerPool {
	rand.Seed(time.Now().UnixNano()) // for shuffling nodePool keys
	return &WorkerPool{
		freePool:     make(map[string]WorkerDetail),
		nodePool:     make(map[string][]WorkerDetail),
		reservedPool: make(map[string]WorkerDetail),
	}
}

func (pool *WorkerPool) Register(addr string, nodename string) error {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Errorf("Invalid address for Register: %w", err)
	}
	if nodename == "" {
		return fmt.Errorf("Node name required for Register")
	}
	wd := WorkerDetail{Addr: addr, NodeName: nodename}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	_, prs := pool.reservedPool[addr]
	if prs {
		return nil // ignore register for existing workers
	}
	pool.freePool[addr] = wd
	_, prs = pool.nodePool[nodename]
	if !prs {
		pool.nodePool[nodename] = make([]WorkerDetail, 0)
	}
	pool.nodePool[nodename] = append(pool.nodePool[nodename], wd)
	return nil
}

// removeFromNodePool is internal and the calling function must be holding the mutex
func (pool *WorkerPool) removeFromNodePool(wd WorkerDetail) {
	s := pool.nodePool[wd.NodeName]
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
	if len(s) == 1 {
		// the convention is to remove empty lists from the hash
		// so len(pool.nodePool) is a good estimate
		delete(pool.nodePool, wd.NodeName)
	} else {
		// overwrite with the last element and truncate slice
		s[i] = s[len(s)-1]
		pool.nodePool[wd.NodeName] = s[:len(s)-1]
	}
}

func (pool *WorkerPool) Deregister(addr string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	wd, prs := pool.freePool[addr]
	if prs {
		pool.removeFromNodePool(wd)
		delete(pool.freePool, addr)
	}
}

// Unreserve removes from the reserved pool, but does not reregister
// We expect the worker process to initiate the register
func (pool *WorkerPool) Unreserve(addr string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	delete(pool.reservedPool, addr)
}

// Recruit attempts to spread the workers across nodes
func (pool *WorkerPool) Recruit(n int) ([]WorkerDetail, error) {
	if n < 1 {
		return nil, fmt.Errorf("Recruit must request one or more workers: n=%d", n)
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	var nodecount = len(pool.nodePool)
	if nodecount < 1 {
		return []WorkerDetail{}, nil
	}

	// Make a single pass through the nodes in the cluster that have
	// available workers, and try to pick evenly from each node. If that pass
	// fails to recruit enough workers, then we start to pick from the freePool, regardless
	// of node, until we recruit enough workers, or all available workers.

	var keys []string
	for k, _ := range pool.nodePool {
		keys = append(keys, k)
	}
	// shuffle the keys
	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	recruits := make([]WorkerDetail, 0)
	for i, key := range keys {
		workers := pool.nodePool[key]
		// adjust goal on each iteration
		goal := int(math.Ceil(float64(n-len(recruits)) / float64(nodecount-i)))
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

	// If there are still recruits needed, select them "randomly"
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

	// Now add the recruits to the Reserved Pool
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
