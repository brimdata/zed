package sort

import (
	"container/heap"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// runManager manages runs (files of sorted records).
type RunManager struct {
	runs       []*runFile
	runIndices map[*runFile]int
	compareFn  expr.CompareFn
	tempDir    string
	zctx       *resolver.Context
}

// NewRunManager creates a temporary directory.  Call Cleanup to remove it.
func NewRunManager(compareFn expr.CompareFn) (*RunManager, error) {
	tempDir, err := ioutil.TempDir("", "zq-sort-")
	if err != nil {
		return nil, err
	}
	return &RunManager{
		runIndices: make(map[*runFile]int),
		compareFn:  compareFn,
		tempDir:    tempDir,
		zctx:       resolver.NewContext(),
	}, nil
}

func (r *RunManager) Cleanup() {
	for _, run := range r.runs {
		run.closeAndRemove()
	}
	os.RemoveAll(r.tempDir)
}

// CreateRun creates a new run containing the records in recs.
func (r *RunManager) CreateRun(recs []*zng.Record) error {
	expr.SortStable(recs, r.compareFn)
	index := len(r.runIndices)
	filename := filepath.Join(r.tempDir, strconv.Itoa(index))
	runFile, err := newRunFile(filename, recs, r.zctx)
	if err != nil {
		return err
	}
	r.runIndices[runFile] = index
	heap.Push(r, runFile)
	return nil
}

// Peek returns the next record without advancing the reader.  The record stops
// being valid at the next read call.
func (r *RunManager) Peek() (*zng.Record, error) {
	if r.Len() == 0 {
		return nil, nil
	}
	return r.runs[0].nextRecord, nil
}

// Read returns the smallest record (per Less) from among the next records in
// each run.  It implements the merge operation for an external merge sort.
func (r *RunManager) Read() (*zng.Record, error) {
	for {
		if r.Len() == 0 {
			return nil, nil
		}
		rec, eof, err := r.runs[0].read()
		if err != nil {
			return nil, err
		}
		if eof {
			r.runs[0].closeAndRemove()
			heap.Pop(r)
		} else {
			heap.Fix(r, 0)
		}
		if rec != nil {
			return rec, nil
		}
	}
}

func (r *RunManager) Len() int { return len(r.runs) }

func (r *RunManager) Less(i, j int) bool {
	v := r.compareFn(r.runs[i].nextRecord, r.runs[j].nextRecord)
	switch {
	case v < 0:
		return true
	case v == 0:
		// Maintain stability.
		return r.runIndices[r.runs[i]] < r.runIndices[r.runs[j]]
	default:
		return false
	}
}

func (r *RunManager) Swap(i, j int) { r.runs[i], r.runs[j] = r.runs[j], r.runs[i] }

func (r *RunManager) Push(x interface{}) { r.runs = append(r.runs, x.(*runFile)) }

func (r *RunManager) Pop() interface{} {
	x := r.runs[len(r.runs)-1]
	r.runs = r.runs[:len(r.runs)-1]
	return x
}
