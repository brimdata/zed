package slicer

import "io"

type Slice struct {
	Offset uint64
	Length uint64
}

func (s Slice) Overlaps(x Slice) bool {
	return x.Offset >= s.Offset && x.Offset < s.Offset+x.Length
}

func (s Slice) NewReader(r io.ReaderAt) *io.SectionReader {
	return io.NewSectionReader(r, int64(s.Offset), int64(s.Length))
}
