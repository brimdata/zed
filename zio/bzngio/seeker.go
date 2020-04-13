package bzngio

import (
	"io"

	"github.com/brimsec/zq/zng/resolver"
)

type Seeker struct {
	Reader
	seeker io.ReadSeeker
}

func NewSeeker(s io.ReadSeeker, zctx *resolver.Context, framesize int) *Seeker {
	return &Seeker{
		Reader: *NewReaderWithSize(s, zctx, framesize),
		seeker: s,
	}
}

func (s *Seeker) Seek(offset int64) (int64, error) {
	s.peeker.Reset()
	s.zctx.Reset()
	n, err := s.seeker.Seek(offset, 0)
	s.position = n
	return n, err
}
