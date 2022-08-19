package seekindex

import (
	"fmt"
	"io"
)

type Range struct {
	Start, End int64
}

func (r Range) TrimEnd(end int64) Range {
	if r.End > end {
		r.End = end
	}
	return r
}

func (r Range) Crop(r2 Range) Range {
	if r.Start < r2.Start {
		r.Start = r2.Start
	}
	if r.End > r2.End {
		r.End = r2.End
	}
	return r
}

func (r Range) Size() int64 {
	return r.End - r.Start
}

func (r Range) IsZero() bool {
	return r.Start == 0 && r.End == 0
}

func (r Range) String() string {
	return fmt.Sprintf("start %d end %d", r.Start, r.End)
}

func (r Range) Reader(reader io.ReaderAt) (io.Reader, error) {
	return io.NewSectionReader(reader, r.Start, r.Size()), nil
}
