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
// merge-sort its outputs. If the inputs are all sorted according to the
// same comparison function, the combiner output will similarly be
// sorted.
type Combiner struct {
	compare expr.CompareFn
	readers []zbuf.Reader
	done    []bool
	hol     []*zng.Record
}

// NewCombiner returns a new combiner
func NewCombiner(readers []zbuf.Reader, c expr.CompareFn) *Combiner {
	return &Combiner{
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

// Peek returns the current minimum record (under the combiner's
// compare function) across all current reader HOLs.
func (c *Combiner) Peek() (*zng.Record, error) {
	rec, _, err := c.peek()
	return rec, err
}

func (c *Combiner) peek() (*zng.Record, int, error) {
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
	rec, i, err := c.peek()
	if rec == nil || err != nil {
		return rec, err
	}
	c.hol[i] = nil
	return rec, nil
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

type Peeker interface {
	zbuf.Reader
	Peek() (*zng.Record, error)
}

// A MergeReader reads records from a Peeker, merging all identical
// records into one upon reading from the combiner, where record
// identity is defined by the compare function.
type MergeReader struct {
	reader  Peeker
	compare expr.CompareFn
	merge   MergeFunc
}

func NewMergeReader(r Peeker, c expr.CompareFn, m MergeFunc) MergeReader {
	return MergeReader{r, c, m}
}

func (m *MergeReader) Read() (*zng.Record, error) {
	rec, err := m.reader.Read()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	recs := []*zng.Record{rec}
	for {
		rec, err := m.reader.Peek()
		if err != nil {
			return nil, err
		}
		if rec == nil || m.compare(recs[0], rec) != 0 {
			break
		}
		recs = append(recs, rec)
		_, _ = m.reader.Read()
	}
	return m.merge(recs[0], recs[1:]...)
}
