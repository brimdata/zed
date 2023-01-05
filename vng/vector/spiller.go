package vector

import (
	"io"

	"github.com/pierrec/lz4/v4"
	"golang.org/x/exp/slices"
)

type Spiller struct {
	writer io.Writer
	Thresh int

	buf        []byte
	compressor lz4.Compressor
	off        int64
}

func NewSpiller(w io.Writer, thresh int) *Spiller {
	return &Spiller{
		writer: w,
		Thresh: thresh,
	}
}

func (s *Spiller) Position() int64 {
	return s.off
}

func (s *Spiller) Write(segments []Segment, b []byte) ([]Segment, error) {
	cf := CompressionFormatNone
	contentLen := len(b)
	// Use contentLen-1 so compression will fail if it doesn't result in
	// fewer bytes.
	s.buf = slices.Grow(s.buf[:0], contentLen-1)[:contentLen-1]
	zlen, err := s.compressor.CompressBlock(b, s.buf)
	if err != nil && err != lz4.ErrInvalidSourceShortBuffer {
		return nil, err
	}
	if zlen > 0 {
		// Compression succeeded.
		b = s.buf[:zlen]
		cf = CompressionFormatLZ4
	}
	if _, err := s.writer.Write(b); err != nil {
		return nil, err
	}
	segment := Segment{s.off, int32(len(b)), int32(contentLen), cf}
	s.off += int64(len(b))
	return append(segments, segment), nil
}
