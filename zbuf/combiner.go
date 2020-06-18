package zbuf

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/zng"
)

type Direction bool

const (
	DirTimeForward = Direction(true)
	DirTimeReverse = Direction(false)
)

func RecordCompare(d Direction) RecordCmpFn {
	if d == DirTimeForward {
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

type Combiner struct {
	readers []Reader
	hol     []*zng.Record
	done    []bool
	less    RecordCmpFn
}

// NewCombiner returns a Combiner that merges zbuf.Readers into
// a single Reader. If the ordering of the input readers matches
// the direction specified, the output records will maintain
// that order.
func NewCombiner(readers []Reader, cmp RecordCmpFn) *Combiner {
	return &Combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
		less:    cmp,
	}
}

func (c *Combiner) Read() (*zng.Record, error) {
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

func (c *Combiner) closeReader(r Reader) error {
	if closer, ok := r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Close closes underlying Readers implementing the io.Closer
// interface if they haven't already been closed.
func (c *Combiner) Close() error {
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
