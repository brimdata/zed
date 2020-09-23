package spill

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

// Merger manages "runs" (files of sorted zng records) that are spilled to
// disk a chunk at a time, then read back and merged in sorted order, effectively
// implementing an external merge sort.
type Merger struct {
	runs       []*peeker
	runIndices map[*peeker]int
	compareFn  expr.CompareFn
	tempDir    string
	zctx       *resolver.Context
}

const TempPrefix = "zq-spill-"

func TempDir() (string, error) {
	return ioutil.TempDir("", TempPrefix)
}

func TempFile() (*os.File, error) {
	return ioutil.TempFile("", TempPrefix)
}

// NewMerger returns Merger to implement external merge sorts of a large
// zng record stream.  It creates a temporary directory to hold the collection
// of spilled chunks.  Call Cleanup to remove it.
func NewMerger(compareFn expr.CompareFn) (*Merger, error) {
	tempDir, err := TempDir()
	if err != nil {
		return nil, err
	}
	return &Merger{
		runIndices: make(map[*peeker]int),
		compareFn:  compareFn,
		tempDir:    tempDir,
		zctx:       resolver.NewContext(),
	}, nil
}

func (r *Merger) Cleanup() {
	for _, run := range r.runs {
		run.closeAndRemove()
	}
	os.RemoveAll(r.tempDir)
}

// Spiil spills a new run of records to a file in the Merger's temp directory.
func (r *Merger) Spill(recs []*zng.Record) error {
	expr.SortStable(recs, r.compareFn)
	index := len(r.runIndices)
	filename := filepath.Join(r.tempDir, strconv.Itoa(index))
	runFile, err := newPeeker(filename, recs, r.zctx)
	if err != nil {
		return err
	}
	r.runIndices[runFile] = index
	heap.Push(r, runFile)
	return nil
}

// Peek returns the next record without advancing the reader.  The record stops
// being valid at the next read call.
func (r *Merger) Peek() (*zng.Record, error) {
	if r.Len() == 0 {
		return nil, nil
	}
	return r.runs[0].nextRecord, nil
}

// Read returns the smallest record (per the comparison function provided to MewMerger)
// from among the next records in the spilled chunks.  It implements the merge operation
// for an external merge sort.
func (r *Merger) Read() (*zng.Record, error) {
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

func (r *Merger) Len() int { return len(r.runs) }

func (r *Merger) Less(i, j int) bool {
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

func (r *Merger) Swap(i, j int) { r.runs[i], r.runs[j] = r.runs[j], r.runs[i] }

func (r *Merger) Push(x interface{}) { r.runs = append(r.runs, x.(*peeker)) }

func (r *Merger) Pop() interface{} {
	x := r.runs[len(r.runs)-1]
	r.runs = r.runs[:len(r.runs)-1]
	return x
}
