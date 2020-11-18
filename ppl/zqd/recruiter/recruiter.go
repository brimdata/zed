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
	FreePool     map[string]WorkerDetail   // Map of all free workers
	NodePool     map[string][]WorkerDetail // Map of nodes of slices of free workers
	ReservedPool map[string]WorkerDetail   // Map of busy workers
}

type WorkerDetail struct {
	Addr     string
	NodeName string
}

func NewWorkerPool() *WorkerPool {
	rand.Seed(time.Now().UnixNano()) // for shuffling NodePool keys
	return &WorkerPool{
		FreePool:     make(map[string]WorkerDetail),
		NodePool:     make(map[string][]WorkerDetail),
		ReservedPool: make(map[string]WorkerDetail),
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
	_, prs := pool.ReservedPool[addr]
	if prs {
		return nil // ignore register for existing workers
	}
	pool.FreePool[addr] = wd
	_, prs = pool.NodePool[nodename]
	if !prs {
		pool.NodePool[nodename] = make([]WorkerDetail, 0)
	}
	pool.NodePool[nodename] = append(pool.NodePool[nodename], wd)
	println(len(pool.FreePool))
	return nil
}

func (pool *WorkerPool) Deregister(addr string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	wd, prs := pool.FreePool[addr]
	if prs {
		// remove from array while not preserving order
		s := pool.NodePool[wd.NodeName]
		i := -1
		for j, v := range s {
			if v == wd {
				i = j
				break
			}
		}
		if i != -1 {
			if len(s) == 1 {
				delete(pool.NodePool, wd.NodeName)
			} else {
				s[len(s)-1], s[i] = s[i], s[len(s)-1]
				pool.NodePool[wd.NodeName] = s[:len(s)-1]
			}
		}
	}
	delete(pool.FreePool, addr)
}

// Unreserve removes from the reserved pool, but does not reregister
// We expect the worker process to initiate the register
func (pool *WorkerPool) Unreserve(addr string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	delete(pool.ReservedPool, addr)
}

func recalcGoal(n int, nodecount int) int {
	return int(math.Ceil(float64(n) / float64(nodecount)))
}

// Recruit attempts to spread the workers across nodes
func (pool *WorkerPool) Recruit(n int) ([]string, error) {
	if n < 1 {
		return nil, fmt.Errorf("Recruit must request one or more workers: n=%d", n)
	}
	pool.mu.Lock()
	defer pool.mu.Unlock()
	var nodecount = len(pool.NodePool)
	if nodecount < 1 {
		return nil, fmt.Errorf("No workers available")
	}

	// Make a single pass through the nodes in the cluster that have
	// available workers, and try to pick evenly from each node. If that pass
	// fails to recruit enough workers, then we start to pick from the FreePool, regardless
	// of node, until we recruit enough workers, or all available workers.

	var keys []string
	for k, _ := range pool.NodePool {
		keys = append(keys, k)
	}
	// shuffle the keys
	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })

	remainingNodes := len(keys)
	recruitsNeeded := n
	goal := recalcGoal(n, nodecount)
	recruits := make([]WorkerDetail, 0)

	for _, key := range keys {
		remainingNodes--
		workers := pool.NodePool[key]
		if goal > recruitsNeeded {
			goal = recruitsNeeded
		}
		if len(workers) > goal {
			recruitsNeeded -= goal
			recruits = append(recruits, workers[:goal]...)
			pool.NodePool[key] = workers[goal:]
		} else {
			recruitsNeeded -= len(workers)
			// not enough recruits from this node, recalculate goal
			goal = recalcGoal(recruitsNeeded, remainingNodes)
			recruits = append(recruits, workers...)
			delete(pool.NodePool, key)
		}
		if recruitsNeeded == 0 {
			break
		}
	}

	// Delete the recruits obtained in this pass from the FreePool
	for _, wd := range recruits {
		delete(pool.FreePool, wd.Addr)
	}

	// If there are still recruits needed, select them "randomly"
	if recruitsNeeded > 0 {
		for k, wd := range pool.FreePool {
			recruits = append(recruits, wd)
			delete(pool.FreePool, k)
			recruitsNeeded--
			if recruitsNeeded < 1 {
				break
			}
		}
	}

	// Now add the recruits to the Reserved Pool
	retval := make([]string, len(recruits))
	for i, r := range recruits {
		pool.ReservedPool[r.Addr] = r
		retval[i] = r.Addr
	}
	return retval, nil
}
