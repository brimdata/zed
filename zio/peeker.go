package zio

import "github.com/brimdata/zed"

// Peeker wraps a Stream while adding a Peek method, which allows inspection
// of the next item to be read without actually reading it.
type Peeker struct {
	Reader
	cache *zed.Value
}

func NewPeeker(reader Reader) *Peeker {
	return &Peeker{Reader: reader}
}

func (p *Peeker) Peek() (*zed.Value, error) {
	var err error
	if p.cache == nil {
		p.cache, err = p.Reader.Read()
	}
	return p.cache, err
}

func (p *Peeker) Read() (*zed.Value, error) {
	v := p.cache
	if v != nil {
		p.cache = nil
		return v, nil
	}
	return p.Reader.Read()
}
