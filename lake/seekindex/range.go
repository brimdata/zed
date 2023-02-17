package seekindex

import (
	"fmt"
	"io"
)

type Range struct {
	Offset int64 `zed:"offset"`
	Length int64 `zed:"length"`
}

func (r Range) IsZero() bool {
	return r.Length == 0
}

func (r Range) String() string {
	return fmt.Sprintf("offset %d length %d", r.Offset, r.Length)
}

func (r Range) Reader(reader io.ReaderAt) (io.Reader, error) {
	return io.NewSectionReader(reader, r.Offset, r.Length), nil
}
