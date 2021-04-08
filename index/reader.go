package index

import (
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

const (
	frameThresh  = 32 * 1024
	FrameFudge   = 1024
	FrameBufSize = frameThresh + FrameFudge
	FrameMaxSize = 20 * 1024 * 1024
)

// Reader implements zbuf.Reader, io.ReadSeeker, and io.Closer.
type Reader struct {
	zngio.Seeker
	reader     iosrc.Reader
	path       iosrc.URI
	zctx       *zson.Context
	size       int64
	trailer    *Trailer
	trailerLen int
}

// NewReader returns a Reader ready to read a microindex.
// Close() should be called when done.  This embeds a zngio.Seeker so
// Seek() may be called on this Reader.  Any call to Seek() must be to
// an offset that begins a new zng stream (e.g., beginning of file or
// the data immediately following an end-of-stream code)
func NewReader(zctx *zson.Context, path string) (*Reader, error) {
	uri, err := iosrc.ParseURI(path)
	if err != nil {
		return nil, err
	}
	return NewReaderFromURI(context.Background(), zctx, uri)
}

func NewReaderWithContext(ctx context.Context, zctx *zson.Context, path string) (*Reader, error) {
	uri, err := iosrc.ParseURI(path)
	if err != nil {
		return nil, err
	}
	return NewReaderFromURI(ctx, zctx, uri)
}

func NewReaderFromURI(ctx context.Context, zctx *zson.Context, uri iosrc.URI) (*Reader, error) {
	r, err := iosrc.NewReader(ctx, uri)
	if err != nil {
		return nil, err
	}
	// Grab the size so we don't seek past the front of the file and
	// cause an error.  XXX this causes an extra synchronous round-trip
	// in the inner loop of a microindex scan, so we might want to do this
	// in parallel with the open either by extending the iosrc interface
	// or running this call here in its own goroutine (before the open)
	si, err := iosrc.Stat(ctx, uri)
	if err != nil {
		return nil, err
	}
	size := si.Size()
	trailer, trailerLen, err := readTrailer(r, size)
	if err != nil {
		r.Close()
		return nil, fmt.Errorf("%s: %w", uri, err)
	}
	if trailer.FrameThresh > FrameMaxSize {
		return nil, fmt.Errorf("%s: frame threshold too large (%d)", uri, trailer.FrameThresh)
	}
	// We add a bit to the seeker buffer so to accommodate the usual
	// overflow size.
	seeker := zngio.NewSeekerWithSize(r, zctx, trailer.FrameThresh+FrameFudge)
	reader := &Reader{
		Seeker:     *seeker,
		reader:     r,
		path:       uri,
		zctx:       zctx,
		size:       size,
		trailer:    trailer,
		trailerLen: trailerLen,
	}
	return reader, nil
}

func (r *Reader) IsEmpty() bool {
	if r.trailer == nil {
		panic("IsEmpty called on a Reader with an error")
	}
	return r.trailer.Sections == nil
}

func (r *Reader) section(level int) (int64, int64) {
	off := int64(0)
	for k := 0; k < level; k++ {
		off += r.trailer.Sections[k]
	}
	return off, r.trailer.Sections[level]
}

func (r *Reader) newSectionReader(level int, sectionOff int64) (zbuf.Reader, error) {
	off, len := r.section(level)
	off += sectionOff
	len -= sectionOff
	reader := io.NewSectionReader(r.reader, off, len)
	return zngio.NewReaderWithOpts(reader, r.zctx, zngio.ReaderOpts{Size: FrameBufSize}), nil
}

func (r *Reader) NewSectionReader(section int) (zbuf.Reader, error) {
	n := len(r.trailer.Sections)
	if section < 0 || section >= n {
		return nil, fmt.Errorf("section %d out of range (index has %d sections)", section, n)
	}
	return r.newSectionReader(section, 0)
}

func (r *Reader) NewTrailerReader() (zbuf.Reader, error) {
	off := r.size - int64(r.trailerLen)
	reader := io.NewSectionReader(r.reader, off, int64(r.trailerLen))
	return zngio.NewReaderWithOpts(reader, r.zctx, zngio.ReaderOpts{Size: r.trailerLen}), nil
}

func (r *Reader) Close() error {
	return r.reader.Close()
}

func (r *Reader) Path() string {
	return r.path.String()
}

func (r *Reader) Order() zbuf.Order {
	return r.trailer.Order
}

func (r *Reader) Keys() *zng.TypeRecord {
	return r.trailer.KeyType
}
