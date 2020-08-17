// This file contains code from these Go 1.14.5 source files:
// - container/heap/example_intheap_test.go
// and is covered by the copyright below.
// The changes are covered by the copyright and license in the
// LICENSE file in the root directory of this repository.

// Copyright (c) 2009 The Go Authors. All rights reserved.
// See acknowledgments.txt for full license text from:
// https://github.com/golang/go/blob/master/LICENSE

package driver

// An intHeap is a min-heap of ints.
type intHeap []int

func (h intHeap) Len() int           { return len(h) }
func (h intHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h intHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *intHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(int))
}

func (h *intHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
