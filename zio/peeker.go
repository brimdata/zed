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

func (p *Peeker) Peek(arena *zed.Arena) (*zed.Value, error) {
	if p.cache != nil {
		p.cache.CheckArena(arena)
		return p.cache, nil
	}
	var err error
	p.cache, err = p.Reader.Read(arena)
	return p.cache, err
}

func (p *Peeker) Read(arena *zed.Arena) (*zed.Value, error) {
	if val := p.cache; val != nil {
		val.CheckArena(arena)
		p.cache = nil
		return val, nil
	}
	return p.Reader.Read(arena)
}
