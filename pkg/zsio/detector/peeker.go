package detector

import (
	"io"
)

type Peeker struct {
	io.Reader
	buffer []byte
}

func NewPeeker(r io.Reader, n int) (*Peeker, error) {
	b := make([]byte, n)
	cc := 0
	for cc < n {
		n, err := r.Read(b[cc:])
		cc += n
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

	}
	return &Peeker{
		Reader: r,
		buffer: b[:cc],
	}, nil
}

func (p *Peeker) Peek() []byte {
	return p.buffer
}

func (p *Peeker) Read(b []byte) (int, error) {
	if p.buffer == nil {
		return p.Reader.Read(b)
	}
	n := len(p.buffer)
	if n > len(b) {
		n = len(b)
	}
	copy(b, p.buffer[:n])
	p.buffer = p.buffer[n:]
	if len(p.buffer) == 0 {
		// no longer needed, return to GC
		p.buffer = nil

	}
	return n, nil
}
