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
	//println(len(pool.freePool))
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
		// this will only happen when there is a bug in this file
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
	var nodecount = len(pool.nodePool)
	if nodecount < 1 {
		return nil, fmt.Errorf("No workers available")
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

	remainingNodes := len(keys)
	recruitsNeeded := n
	goal := recalcGoal(n, nodecount)
	recruits := make([]WorkerDetail, 0)

	for _, key := range keys {
		remainingNodes--
		workers := pool.nodePool[key]
		if goal > recruitsNeeded {
			goal = recruitsNeeded
		}
		if len(workers) > goal {
			recruitsNeeded -= goal
			recruits = append(recruits, workers[:goal]...)
			pool.nodePool[key] = workers[goal:]
		} else {
			recruitsNeeded -= len(workers)
			// not enough recruits from this node, recalculate goal
			goal = recalcGoal(recruitsNeeded, remainingNodes)
			recruits = append(recruits, workers...)
			delete(pool.nodePool, key)
		}
		if recruitsNeeded == 0 {
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
		//println("deleted ", wd.Addr)
	}

	// If there are still recruits needed, select them "randomly"
	if recruitsNeeded > 0 {
		for k, wd := range pool.freePool {
			pool.removeFromNodePool(wd)
			delete(pool.freePool, k)
			recruits = append(recruits, wd)
			recruitsNeeded--
			if recruitsNeeded < 1 {
				break
			}
		}
	}

	// Now add the recruits to the Reserved Pool
	retval := make([]string, len(recruits))
	for i, r := range recruits {
		pool.reservedPool[r.Addr] = r
		retval[i] = r.Addr
	}
	return retval, nil
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
