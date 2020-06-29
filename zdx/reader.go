package zdx

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
)

const (
	FrameSize = 32 * 1024
)

// Reader implements zbuf.Reader, io.ReadSeeker, and io.Closer.
type Reader struct {
	zngio.Seeker
	reader io.ReadCloser
}

// NewReader returns a Reader ready to read a zdx.
// Close() should be called when done.  This embeds a bnzgio.Seeker so
// Seek() may be called on this Reader.  Any call to Seek() must be to
// an offset that begins a new zng stream (e.g., beginning of file or
// the data immediately following an end-of-stream code)
func NewReader(zctx *resolver.Context, path string) (*Reader, error) {
	uri, err := iosrc.ParseURI(path)
	if err != nil {
		return nil, err
	}
	return newReader(zctx, uri, 0)
}

func newReader(zctx *resolver.Context, uri iosrc.URI, level int) (*Reader, error) {
	r, err := iosrc.NewReader(filename(uri, level))
	if err != nil {
		return nil, err
	}
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("underyling iosrc.NewReader did not return an io.ReadSeeker")
	}
	seeker := zngio.NewSeekerWithSize(rs, zctx, FrameSize)
	return &Reader{
		Seeker: *seeker,
		reader: r,
	}, nil
}

func (r *Reader) Close() error {
	return r.reader.Close()
}
