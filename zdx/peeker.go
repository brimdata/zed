package zdx

// Peeker wraps a Stream while adding a Peek method, which allows inspection
// of the next item to be read without actually reading it.
type Peeker struct {
	Stream
	cache Pair
}

func NewPeeker(s Stream) *Peeker {
	return &Peeker{Stream: s}
}

func (p *Peeker) Peek() (Pair, error) {
	if p.cache.Key == nil {
		var err error
		p.cache, err = p.Stream.Read()
		if err != nil {
			return Pair{}, err
		}
	}
	return p.cache, nil
}

func (p *Peeker) Read() (Pair, error) {
	v := p.cache
	if v.Key != nil {
		p.cache = Pair{}
		return v, nil
	}
	return p.Stream.Read()
}
