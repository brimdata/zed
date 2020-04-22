package zdx

import (
	"bytes"
	"io"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

//XXX this is looking a lot like scanner.Combiner expect we're using
// keys instead of timestamps to sort.  Maybe we should merge these
// two modules.

type MergeFunc func(*zng.Record, *zng.Record) *zng.Record

type record struct {
	rec *zng.Record
	key zcode.Bytes
}

// Combiner reads from two or more Streams, implementing a new Stream from
// the merged streams, while preserving the lexicographic order of the keys.
// It calls the combine function to merge values that have the same key.
type Combiner struct {
	combine MergeFunc
	streams []zbuf.Reader
	done    []bool
	hol     []record
}

// NewCombiner returns a new combiner
func NewCombiner(streams []zbuf.Reader, f MergeFunc) *Combiner {
	return &Combiner{
		combine: f,
		streams: streams,
		hol:     make([]record, len(streams)),
		done:    make([]bool, len(streams)),
	}
}

func (c *Combiner) Read() (*zng.Record, error) {
	// XXX if this gets big, we can optimize by creating a mergesort tree...
	var minkey zcode.Bytes
	for k := range c.streams {
		if c.done[k] {
			continue
		}
		if c.hol[k].rec == nil {
			rec, err := c.streams[k].Read()
			if err != nil {
				return nil, err
			}
			if rec == nil {
				c.done[k] = true
				continue
			}
			// cache the slice lookup to make the logic simpler
			key, err := rec.Slice(0)
			if err != nil {
				return nil, err
			}
			c.hol[k] = record{rec, key}
		}
		if minkey == nil || bytes.Compare(c.hol[k].key, minkey) < 0 {
			minkey = c.hol[k].key
		}
	}
	if minkey == nil {
		return nil, nil
	}
	// We found the smallest key of the head-of-line records.
	// Now spin through the slice and find either the sole record
	// that has the smallest key or combine records that have the same
	// key using the custom merge function.  The trivial merge function,
	// where there are no values, will just keep returning the first
	// record without any allocs which has the desired effect of
	// deduplicating the keys for key-only tables.
	var rec *zng.Record
	for k := range c.streams {
		if c.hol[k].rec != nil && bytes.Equal(minkey, c.hol[k].key) {
			if rec == nil {
				rec = c.hol[k].rec
			} else {
				// XXX this could be more efficient if we bundle
				// multiple values to be combined all at once
				// in the client domain rather than converting
				// to/from byte slice
				rec = c.combine(rec, c.hol[k].rec)
			}
			c.hol[k].rec = nil
		}
	}
	return rec, nil
}

//XXX maybe this should be left to the caller?
func (c *Combiner) Close() error {
	var lastErr error
	for _, r := range c.streams {
		if closer, ok := r.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}
