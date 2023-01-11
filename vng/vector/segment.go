package vector

import (
	"fmt"
	"io"
	"sync"

	"github.com/pierrec/lz4/v4"
	"golang.org/x/exp/slices"
)

// Values for [Segment.CompressionFormat].
const (
	CompressionFormatNone uint8 = 0 // No compression
	CompressionFormatLZ4  uint8 = 1 // LZ4 compression
)

type Segment struct {
	Offset            int64 // Offset relative to start of file
	Length            int32 // Length in file
	MemLength         int32 // Length in memory
	CompressionFormat uint8 // Compression format in file
}

var zbufPool = sync.Pool{
	New: func() any { return new([]byte) },
}

// Read reads the segement r, uncompresses it if necessary, and stores it in the
// first s.MemLength bytes of b. If the length of b is less than s.MemLength,
// Read returns [io.ErrShortBuffer].
func (s *Segment) Read(r io.ReaderAt, b []byte) error {
	if len(b) < int(s.MemLength) {
		return io.ErrShortBuffer
	}
	b = b[:s.MemLength]
	switch s.CompressionFormat {
	case CompressionFormatNone:
		_, err := r.ReadAt(b, s.Offset)
		return err
	case CompressionFormatLZ4:
		zbuf := zbufPool.Get().(*[]byte)
		defer zbufPool.Put(zbuf)
		*zbuf = slices.Grow((*zbuf)[:0], int(s.Length))[:s.Length]
		if _, err := r.ReadAt(*zbuf, s.Offset); err != nil {
			return err
		}
		n, err := lz4.UncompressBlock(*zbuf, b)
		if err != nil {
			return err
		}
		if n != int(s.MemLength) {
			return fmt.Errorf("vng: got %d uncompressed bytes, expected %d", n, s.MemLength)
		}
		return nil
	default:
		return fmt.Errorf("vng: unknown compression format 0x%x", s.CompressionFormat)
	}
}
