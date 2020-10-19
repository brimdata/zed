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

func RecordCompare(o Order) RecordCmpFn {
	if o == OrderAsc {
		return CmpTimeForward
	}
	return CmpTimeReverse
}

// RecordCmpFn returns true if a < b.
type RecordCmpFn func(a, b *zng.Record) bool

func CmpTimeForward(a, b *zng.Record) bool {
	return a.Ts() < b.Ts()
}

func CmpTimeReverse(a, b *zng.Record) bool {
	return !CmpTimeForward(a, b)
}

// NewCombiner returns a ReaderCloser that merges zbuf.Readers into
// a single Reader. If the ordering of the input readers matches
// the direction specified, the output records will maintain
// that order.
func NewCombiner(readers []Reader, cmp RecordCmpFn) ReadCloser {
	if len(readers) == 1 {
		if rc, ok := readers[0].(ReadCloser); ok {
			return rc
		}
	}
	return &combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
		less:    cmp,
	}
}

type combiner struct {
	hol     []*zng.Record
	less    RecordCmpFn
	readers []Reader

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
