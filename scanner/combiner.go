package scanner

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Combiner struct {
	readers []zbuf.Reader
	hol     []*zng.Record
	done    []bool
}

func NewCombiner(readers []zbuf.Reader) *Combiner {
	return &Combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
	}
}

func OpenFiles(zctx *resolver.Context, paths ...string) (*Combiner, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		reader, err := OpenFile(zctx, path, "auto")
		if err != nil {
			return nil, err
		}
		readers = append(readers, reader)
	}
	return NewCombiner(readers), nil
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
				if err := c.closeReader(l); err != nil {
					return nil, fmt.Errorf("%s: %w", c.readers[k], err)
				}
				continue
			}
			c.hol[k] = tup
		}
		if idx == -1 || c.hol[k].Ts < c.hol[idx].Ts {
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

func (c *Combiner) closeReader(r zbuf.Reader) error {
	if closer, ok := r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// Close closes underlying zbuf.Readers implementing the io.Closer
// interface if they haven't already been closed.
func (c *Combiner) Close() error {
	var err error
	for k, r := range c.readers {
		if c.done[k] {
			continue
		}
		c.done[k] = true
		// Return only the first error, but closing everything else if there is
		// an error.
		if e := c.closeReader(r); err == nil {
			err = e
		}
	}
	return err
}
