package zio

import (
	"context"
	"slices"
	"sync"

	"github.com/brimdata/zed"
)

// A Combiner is a Reader that returns records by reading from multiple Readers.
type Combiner struct {
	cancel  context.CancelFunc
	ctx     context.Context
	done    []bool
	once    sync.Once
	readers []Reader
	results chan combinerResult
}

func NewCombiner(ctx context.Context, readers []Reader) *Combiner {
	ctx, cancel := context.WithCancel(ctx)
	return &Combiner{
		cancel:  cancel,
		ctx:     ctx,
		done:    make([]bool, len(readers)),
		readers: readers,
		results: make(chan combinerResult),
	}
}

type combinerResult struct {
	err error
	idx int
	val *zed.Value
}

func (c *Combiner) run() {
	for i := range c.readers {
		idx := i
		go func() {
			for {
				rec, err := c.readers[idx].Read()
				if rec != nil {
					// Make a copy since we don't wait for
					// Combiner.Read's caller to finish with
					// this value before we read the next.
					rec = rec.Copy().Ptr()
				}
				select {
				case c.results <- combinerResult{err, idx, rec}:
					if rec == nil || err != nil {
						return
					}
				case <-c.ctx.Done():
					return
				}
			}
		}()
	}
}

func (c *Combiner) finished() bool {
	return !slices.Contains(c.done, false)
}

func (c *Combiner) Read() (*zed.Value, error) {
	c.once.Do(c.run)
	for {
		select {
		case r := <-c.results:
			if r.err != nil {
				c.cancel()
				return nil, r.err
			}
			if r.val != nil {
				return r.val, nil
			}
			c.done[r.idx] = true
			if c.finished() {
				c.cancel()
				return nil, nil
			}
		case <-c.ctx.Done():
			return nil, c.ctx.Err()
		}
	}
}
