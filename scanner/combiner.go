package scanner

import (
	"fmt"

	"github.com/mccanne/zq/pkg/zng"
)

type Reader struct {
	zng.Reader
	Name string
}

type Combiner struct {
	readers []Reader
	hol     []*zng.Record
	done    []bool
}

func NewCombiner(readers []Reader) *Combiner {
	return &Combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
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
				return nil, fmt.Errorf("%s: %w", c.readers[k].Name, err)
			}
			if tup == nil {
				c.done[k] = true
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
