package scanner

import (
	"github.com/mccanne/zq/pkg/zq"
)

type Combiner struct {
	readers []zq.Reader
	hol     []*zq.Record
	done    []bool
}

func NewCombiner(readers []zq.Reader) *Combiner {
	return &Combiner{
		readers: readers,
		hol:     make([]*zq.Record, len(readers)),
		done:    make([]bool, len(readers)),
	}
}

func (c *Combiner) Read() (*zq.Record, error) {
	idx := -1
	for k, l := range c.readers {
		if c.done[k] {
			continue
		}
		if c.hol[k] == nil {
			tup, err := l.Read()
			if err != nil {
				return nil, err
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
