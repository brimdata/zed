package zbuf

import (
	"context"
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
)

type Reader interface {
	Read() (*zng.Record, error)
}

type Writer interface {
	Write(*zng.Record) error
}

type WriteCloser interface {
	Writer
	io.Closer
}

type WriteFlusher interface {
	Writer
	Flush() error
}

type nopFlusher struct {
	Writer
}

func (nopFlusher) Flush() error { return nil }

// NopFlusher returns a WriteFlusher with a no-op Flush method wrapping
// the provided Writer w.
func NopFlusher(w Writer) WriteFlusher {
	return nopFlusher{w}
}

type namedReader struct {
	Reader
	name string
}

func (n namedReader) String() string {
	return fmt.Sprintf("reader<%s>", n.name)
}

func NamedReader(r Reader, name string) Reader {
	return namedReader{r, name}
}

// Batch is an inteface to a bundle of Records.
// Batches can be shared across goroutines via reference counting and should be
// copied on modification when the reference count is greater than 1.
type Batch interface {
	Ref()
	Unref()
	Index(int) *zng.Record
	Length() int
	Records() []*zng.Record
	//XXX span should go in here?
	Span() nano.Span
}

func CopyWithContext(ctx context.Context, dst WriteFlusher, src Reader) error {
	var err error
	for ctx.Err() == nil {
		var rec *zng.Record
		rec, err = src.Read()
		if err != nil || rec == nil {
			break
		}
		err = dst.Write(rec)
		if err != nil {
			break
		}
	}
	dstErr := dst.Flush()
	switch {
	case err != nil:
		return err
	case dstErr != nil:
		return dstErr
	default:
		return ctx.Err()
	}
}

// Copy copies src to dst a la io.Copy.  The src reader is read from
// while the dst writer is written to and closed.
func Copy(dst WriteFlusher, src Reader) error {
	return CopyWithContext(context.Background(), dst, src)
}
