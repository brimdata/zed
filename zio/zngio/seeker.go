package zngio

import (
	"io"

	"github.com/brimdata/zed/zng/resolver"
)

type Seeker struct {
	Reader
	seeker io.ReadSeeker
}

func NewSeeker(s io.ReadSeeker, zctx *resolver.Context) *Seeker {
	return NewSeekerWithSize(s, zctx, ReadSize)
}

func NewSeekerWithSize(s io.ReadSeeker, zctx *resolver.Context, framesize int) *Seeker {
	return &Seeker{
		Reader: *NewReaderWithOpts(s, zctx, ReaderOpts{Size: framesize}),
		seeker: s,
	}
}

func (s *Seeker) Seek(offset int64) (int64, error) {
	s.peeker.Reset()
	s.uncompressedBuf = nil
	s.zctx.Reset()
	n, err := s.seeker.Seek(offset, io.SeekStart)
	s.peekerOffset = n
	return n, err
}
