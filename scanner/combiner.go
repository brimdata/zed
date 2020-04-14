package scanner

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Combiner struct {
	readers  []zbuf.Reader
	hol      []*zng.Record
	done     []bool
	warnings chan string
	stopErr  bool
}

// NewCombiner returns a Combiner from a slice of readers. If stopErr
// is true, the combiner stops reading from all readers after
// encountering an error from any reader. Otherwise, it discards the
// errored reader and keeps reading from the others.
func NewCombiner(readers []zbuf.Reader, stopErr bool) *Combiner {
	return &Combiner{
		readers: readers,
		hol:     make([]*zng.Record, len(readers)),
		done:    make([]bool, len(readers)),
		stopErr: stopErr,
	}
}

// Set a warnings channel, similar to the one in proc.Context. If this
// channel is set, read errors are sent on it as warnings and reading
// from the remaining readers continues. Otherwise, a read error is
// returned by Read().
func (c *Combiner) SetWarningsChan(ch chan string) {
	c.warnings = ch
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
	return NewCombiner(readers, true), nil
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
				msg := fmt.Errorf("%s: %w", c.readers[k], err)
				if c.stopErr {
					return nil, msg
				}
				c.done[k] = true
				if c.warnings != nil {
					c.warnings <- msg.Error()
				}
				continue
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
		c.done[k] = true
		// Return only the first error, but closing everything else if there is
		// an error.
		if e := c.closeReader(r); err == nil {
			err = e
		}
	}
	return err
}
