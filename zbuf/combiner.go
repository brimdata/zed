package zbuf

import (
	"fmt"
	"io"
	"sync"

	"github.com/brimsec/zq/zng"
)

type Order bool

const (
	OrderAsc  = Order(false)
	OrderDesc = Order(true)
)

func (o Order) Int() int {
	if o {
		return -1
	}
	return 1
}

func (o Order) String() string {
	if o {
		return "descending"
	}
	return "ascending"
}

func (o Order) RecordLess() RecordLessFn {
	if o == OrderAsc {
		return RecordLessTsForward
	}
	return RecordLessTsReverse
}

// RecordLessFn returns true if a < b.
type RecordLessFn func(a, b *zng.Record) bool

func RecordLessTsForward(a, b *zng.Record) bool {
	return a.Ts() < b.Ts()
}

func RecordLessTsReverse(a, b *zng.Record) bool {
	return !RecordLessTsForward(a, b)
}

// NewCombiner returns a ReaderCloser that merges zbuf.Readers into
// a single Reader. If the ordering of the input readers matches
// the direction specified, the output records will maintain
// that order.
func NewCombiner(readers []Reader, less RecordLessFn) ReadCloser {
	if len(readers) == 1 {
		if rc, ok := readers[0].(ReadCloser); ok {
			return rc
		}
	}
	return &combiner{
		readers: readers,
		less:    less,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
	}
}

type combiner struct {
	readers []Reader
	less    RecordLessFn
	hol     []*zng.Record

	mu   sync.Mutex // protects below
	done []bool
}

func (c *combiner) Read() (*zng.Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	idx := -1
	for k, l := range c.readers {
		if c.done[k] {
			continue
		}
		if c.hol[k] == nil {
			tup, err := l.Read()
			if err != nil {
				return nil, fmt.Errorf("%s: %w", c.readers[k], err)
			}
			if tup == nil {
				c.done[k] = true
				continue
			}
			c.hol[k] = tup
		}
		if idx == -1 || c.less(c.hol[k], c.hol[idx]) {
			idx = k
		}
	}
	if idx == -1 {
		return nil, nil
	}
	tup := c.hol[idx]
	c.hol[idx] = nil
	return tup, nil
}

func (c *combiner) closeReader(r Reader) error {
	if closer, ok := r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Close closes underlying Readers implementing the io.Closer
// interface if they haven't already been closed.
func (c *combiner) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	for k, r := range c.readers {
		c.done[k] = true
		// Return only the first error, but closing everything else if there is
		// an error.
		if e := c.closeReader(r); err == nil {
			err = e
		}
	}
	return err
}
