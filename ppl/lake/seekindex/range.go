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

func (r Range) LimitReader(reader io.ReadSeeker) (io.Reader, error) {
	if r.Start > 0 {
		if _, err := reader.Seek(r.Start, io.SeekStart); err != nil {
			return nil, err
		}
	}
	return io.LimitReader(reader, r.Size()), nil
}
