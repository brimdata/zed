package zdx

import (
	"os"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
)

const (
	FrameSize = 32 * 1024
)

// Reader implements zbuf.Reader, io.ReadSeeker, and io.Closer.
type Reader struct {
	zngio.Seeker
	file *os.File
}

// NewReader returns a Reader ready to read a zdx.
// Close() should be called when done.  This embeds a bnzgio.Seeker so
// Seek() may be called on this Reader.  Any call to Seek() must be to
// an offset that begins a new zng stream (e.g., beginning of file or
// the data immediately following an end-of-stream code)
func NewReader(zctx *resolver.Context, path string) (*Reader, error) {
	return newReader(zctx, path, 0)
}

func newReader(zctx *resolver.Context, path string, level int) (*Reader, error) {
	f, err := fs.Open(filename(path, level))
	if err != nil {
		return nil, err
	}
	seeker := zngio.NewSeekerWithSize(f, zctx, FrameSize)
	return &Reader{
		Seeker: *seeker,
		file:   f,
	}, nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}
