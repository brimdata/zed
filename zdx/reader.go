package zdx

import (
	"os"

	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng/resolver"
)

const (
	FrameSize = 32 * 1024
)

// Reader implements zbuf.Reader and provides a way too scan zng zdx files as
// well as seek to an offset and read from there.
type Reader struct {
	*bzngio.Seeker
	file *os.File
}

// NewReader returns a Reader ready to read a zdx.
// Close() should be called when done.  This embeds a bnzgio.Seeker so
// Seek() may be called on this Reader.  Any call to Seek() must be to
// an offset that begins a new zng stream (e.g., beginning of file or
// the data immediately following an end-of-stream code)
func NewReader(path string) (*Reader, error) {
	return newReader(path, 0)
}

func newReader(path string, level int) (*Reader, error) {
	f, err := os.Open(filename(path, level))
	if err != nil {
		return nil, err
	}
	seeker := bzngio.NewSeekerWithSize(f, resolver.NewContext(), FrameSize)
	return &Reader{
		Seeker: seeker,
		file:   f,
	}, nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}
