package proc

import (
	"io"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type MergeFunc func(*zng.Record, ...*zng.Record) (*zng.Record, error)

// Combiner reads from two or more sorted inputs, implementing a new
// zbuf.Reader from the inputs. It uses a comparison function to
// merge-sort its outputs. If inputs are all sorted according to the
// same comparison function, the combiner output will similarly be
// sorted.
//
// If the combiner's MergeFunc is non-nil, that function is invoked to
// merge all identical records into one upon reading from the
// combiner, where record identity is defined by the comparison
// function. If it is nil, then a default merger is used that returns
// the first record of each identical set. This can be used to
// deduplicate records.
type Combiner struct {
	merge   MergeFunc
	compare expr.CompareFn
	readers []zbuf.Reader
	done    []bool
	hol     []*zng.Record
}

// NewCombiner returns a new combiner
func NewCombiner(readers []zbuf.Reader, c expr.CompareFn, m MergeFunc) *Combiner {
	if m == nil {
		m = func(r *zng.Record, _ ...*zng.Record) (*zng.Record, error) { return r, nil }
	}
	return &Combiner{
		merge:   m,
		compare: c,
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
	}
}

// Readers can be dynamically added during combiner operation. It is
// the client's responsibility to ensure that the new reader's records
// are all greater than or equal to the biggest record of the existing
// readers.
func (c *Combiner) AddReader(r zbuf.Reader) {
	c.readers = append(c.readers, r)
	c.done = append(c.done, false)
	c.hol = append(c.hol, nil)
}

// PeekMin returns the current minimum record (under the combiner's
// compare function) across all current reader HOLs.
func (c *Combiner) PeekMin() (*zng.Record, error) {
	minrec, _, err := c.peekmin()
	return minrec, err
}

func (c *Combiner) peekmin() (*zng.Record, int, error) {
	var minrec *zng.Record
	var mink int
	for k := range c.readers {
		if c.done[k] {
			continue
		}
		if c.hol[k] == nil {
			rec, err := c.readers[k].Read()
			if err != nil {
				return nil, 0, err
			}
			if rec == nil {
				c.done[k] = true
				continue
			}
			c.hol[k] = rec
		}
		if minrec == nil || c.compare(minrec, c.hol[k]) > 0 {
			minrec = c.hol[k]
			mink = k
		}
	}
	return minrec, mink, nil
}

func (c *Combiner) Read() (*zng.Record, error) {
	minrec, mink, err := c.peekmin()
	if minrec == nil {
		return nil, err
	}
	recs := []*zng.Record{minrec}
	// Assemble a slice with all head-of-line records that are
	// identical to minrec under c.compare, merge them, and return
	// the result.
	for k := range c.readers {
		if k == mink {
			c.hol[k] = nil
			continue
		}
		if c.hol[k] != nil && c.compare(recs[0], c.hol[k]) == 0 {
			recs = append(recs, c.hol[k])
			c.hol[k] = nil
		}
	}
	return c.merge(recs[0], recs[1:]...)
}

func (c *Combiner) Close() error {
	var lastErr error
	for _, r := range c.readers {
		if err := closeReader(r); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func closeReader(r zbuf.Reader) error {
	if closer, ok := r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
