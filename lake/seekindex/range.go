package seekindex

import "io"

type Range struct {
	Start, End int64
}

func (r Range) TrimEnd(end int64) Range {
	if r.End > end {
		r.End = end
	}
	return r
}

func (r Range) Size() int64 {
	return r.End - r.Start
}

func (r Range) Reader(reader io.ReaderAt) (io.Reader, error) {
	return io.NewSectionReader(reader, r.Start, r.Size()), nil
}
