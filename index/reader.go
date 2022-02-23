package index

import (
	"context"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
)

const (
	frameThresh  = 32 * 1024
	FrameFudge   = 1024
	FrameBufSize = frameThresh + FrameFudge
	FrameMaxSize = 20 * 1024 * 1024
)

type Reader struct {
	reader   storage.Reader
	uri      *storage.URI
	zctx     *zed.Context
	size     int64
	meta     FileMeta
	sections []int64
}

var _ io.Closer = (*Reader)(nil)

// NewReader returns a Reader ready to read a microindex.
// Close() should be called when done.  This embeds a zngio.Seeker so
// Seek() may be called on this Reader.  Any call to Seek() must be to
// an offset that begins a new zng stream (e.g., beginning of file or
// the data immediately following an end-of-stream code)
func NewReader(zctx *zed.Context, engine storage.Engine, path string) (*Reader, error) {
	uri, err := storage.ParseURI(path)
	if err != nil {
		return nil, err
	}
	return NewReaderFromURI(context.Background(), zctx, engine, uri)
}

func NewReaderWithContext(ctx context.Context, zctx *zed.Context, engine storage.Engine, path string) (*Reader, error) {
	uri, err := storage.ParseURI(path)
	if err != nil {
		return nil, err
	}
	return NewReaderFromURI(ctx, zctx, engine, uri)
}

func NewReaderFromURI(ctx context.Context, zctx *zed.Context, engine storage.Engine, uri *storage.URI) (*Reader, error) {
	r, err := engine.Get(ctx, uri)
	if err != nil {
		return nil, err
	}
	// Grab the size so we don't seek past the front of the file and
	// cause an error.  XXX this causes an extra synchronous round-trip
	// in the inner loop of a microindex scan, so we might want to do this
	// in parallel with the open either by extending the storage interface
	// or running this call here in its own goroutine (before the open)
	size, err := engine.Size(ctx, uri)
	if err != nil {
		return nil, err
	}
	meta, sections, err := readTrailer(r, size)
	if err != nil {
		r.Close()
		return nil, fmt.Errorf("%s: %w", uri, err)
	}
	if meta.FrameThresh > FrameMaxSize {
		return nil, fmt.Errorf("%s: frame threshold too large (%d)", uri, meta.FrameThresh)
	}
	reader := &Reader{
		reader:   r,
		uri:      uri,
		zctx:     zctx,
		size:     size,
		meta:     *meta,
		sections: sections,
	}
	return reader, nil
}

func (r *Reader) IsEmpty() bool {
	return r.sections == nil
}

func (r *Reader) section(level int) (int64, int64) {
	off := int64(0)
	for k := 0; k < level; k++ {
		off += r.sections[k]
	}
	return off, r.sections[level]
}

func (r *Reader) newSectionReader(level int, sectionOff int64) (*zngio.Reader, error) {
	off, len := r.section(level)
	off += sectionOff
	len -= sectionOff
	sectionReader := io.NewSectionReader(r.reader, off, len)
	return zngio.NewReaderWithOpts(sectionReader, r.zctx, zngio.ReaderOpts{Size: FrameBufSize}), nil
}

func (r *Reader) NewSectionReader(section int) (*zngio.Reader, error) {
	n := len(r.sections)
	if section < 0 || section >= n {
		return nil, fmt.Errorf("section %d out of range (index has %d sections)", section, n)
	}
	return r.newSectionReader(section, 0)
}

func (r *Reader) Close() error {
	return r.reader.Close()
}

func (r *Reader) Path() string {
	return r.uri.String()
}

func (r *Reader) Order() order.Which {
	return r.meta.Order
}

func (r *Reader) Keys() field.List {
	return r.meta.Keys
}
