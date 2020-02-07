package scanner

import (
	"fmt"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Reader struct {
	zbuf.Reader
	Name string
}

type Combiner struct {
	readers []Reader
	hol     []*zng.Record
	done    []bool
	mappers []*resolver.Mapper
}

func NewCombiner(readers []Reader) *Combiner {
	c := &Combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
	}
	n := len(readers)
	c.mappers = make([]*resolver.Mapper, n)
	zctx := resolver.NewContext()
	for k := 0; k < n; k++ {
		c.mappers[k] = resolver.NewMapper(zctx)
	}
	return c
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
			mapper := c.mappers[k]
			id := tup.Type.ID()
			sharedType := mapper.Map(id)
			if sharedType == nil {
				sharedType = mapper.Enter(id, tup.Type)
			}
			tup.Type = sharedType
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
