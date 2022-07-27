package vector

import (
	"io"

	"github.com/brimdata/zed/zcode"
)

type Spiller struct {
	writer io.Writer
	off    int64
	Thresh int
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

func (s *Spiller) Write(segments []Segment, b zcode.Bytes) ([]Segment, error) {
	n, err := s.writer.Write(b)
	if err != nil {
		return segments, err
	}
	segment := Segment{s.off, int32(n)}
	s.off += int64(n)
	return append(segments, segment), nil
}
