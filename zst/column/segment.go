package column

import (
	"io"
)

type Segment struct {
	Offset int64
	Length int32
}

func (s Segment) NewSectionReader(r io.ReaderAt) io.Reader {
	return io.NewSectionReader(r, s.Offset, int64(s.Length))
}
