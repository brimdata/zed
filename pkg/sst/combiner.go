package sst

import (
	"bytes"
)

type Combine func([]byte, []byte) []byte

// Combiner reads from two or more Streams, implementing a new Stream from
// the merged streams, while preserving the lexicographic order of the keys.
// It calls the combine function to merge values that have the same key.
type Combiner struct {
	combine func([]byte, []byte) []byte
	streams []Stream
	done    []bool
	hol     []Pair // head of line for each stream
}

// NewCombiner returns a new combiner
func NewCombiner(streams []Stream, f Combine) *Combiner {
	return &Combiner{
		combine: f,
		streams: streams,
		hol:     make([]Pair, len(streams)),
		done:    make([]bool, len(streams)),
	}
}

func (c *Combiner) Open() error {
	for k, s := range c.streams {
		if err := s.Open(); err != nil {
			return err
		}
		c.hol[k].Value = nil
		c.done[k] = false
	}
	return nil
}

func (c *Combiner) Read() (Pair, error) {
	// XXX if this gets big, we can optimize by creating a mergesort tree...
	var minkey []byte
	for k := range c.streams {
		if c.done[k] {
			continue
		}
		if c.hol[k].Value == nil {
			pair, err := c.streams[k].Read()
			if err != nil {
				return Pair{}, err
			}
			if pair.Value == nil {
				c.done[k] = true
				continue
			}
			c.hol[k] = pair
		}
		if minkey == nil || bytes.Compare(c.hol[k].Key, minkey) < 0 {
			minkey = c.hol[k].Key
		}
	}
	if minkey == nil {
		return Pair{}, nil
	}
	var val []byte
	for k := range c.streams {
		if c.hol[k].Value != nil && bytes.Equal(minkey, c.hol[k].Key) {
			if val == nil {
				val = c.hol[k].Value
			} else {
				// XXX this could be more efficient if we bundle
				// multiple values to be combined all at once
				// in the client domain rather than converting
				// to/from byte slice
				val = c.combine(val, c.hol[k].Value)
			}
			c.hol[k].Value = nil
		}
	}
	return Pair{minkey, val}, nil
}

func (c *Combiner) Close() error {
	for _, s := range c.streams {
		if err := s.Close(); err != nil {
			return err
		}
	}
	return nil
}
