package zbuf

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

type Direction bool

const (
	DirTimeForward = Direction(true)
	DirTimeReverse = Direction(false)
)

type Combiner struct {
	readers []Reader
	hol     []*zng.Record
	done    []bool
	dir     Direction
}

// NewCombiner returns a Combiner that merges zbuf.Readers into
// a single Reader. If the ordering of the input readers matches
// the direction specified, the output records will maintain
// that order.
func NewCombiner(readers []Reader, dir Direction) *Combiner {
	return &Combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
		dir:     dir,
	}
}

func (c *Combiner) Read() (*zng.Record, error) {
	idx := -1
	var cmp func(x, y nano.Ts) bool
	if c.dir == DirTimeForward {
		cmp = func(x, y nano.Ts) bool { return x < y }
	} else {
		cmp = func(x, y nano.Ts) bool { return x > y }
	}
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
		if idx == -1 || cmp(c.hol[k].Ts, c.hol[idx].Ts) {
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
