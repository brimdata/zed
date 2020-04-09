package zdx

import "sync/atomic"

// Counter wraps a Stream and provides a method to return the number
// of Pairs read from the stream.
type Counter struct {
	Stream
	counter *int64
}

func NewCounter(s Stream, p *int64) Stream {
	return &Counter{Stream: s, counter: p}
}

func (c *Counter) Read() (Pair, error) {
	p, err := c.Stream.Read()
	if p.Key != nil {
		atomic.AddInt64(c.counter, 1)
	}
	return p, err
}
