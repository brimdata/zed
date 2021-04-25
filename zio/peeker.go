package zio

import "github.com/brimdata/zed/zng"

// Peeker wraps a Stream while adding a Peek method, which allows inspection
// of the next item to be read without actually reading it.
type Peeker struct {
	Reader
	cache *zng.Record
}

func NewPeeker(reader Reader) *Peeker {
	return &Peeker{Reader: reader}
}

func (p *Peeker) Peek() (*zng.Record, error) {
	var err error
	if p.cache == nil {
		p.cache, err = p.Reader.Read()
	}
	return p.cache, err
}

func (p *Peeker) Read() (*zng.Record, error) {
	v := p.cache
	if v != nil {
		p.cache = nil
		return v, nil
	}
	return p.Reader.Read()
}
