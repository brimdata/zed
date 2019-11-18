package zson

import (
	"context"
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/nano"
)

var (
	ErrDescriptorExists  = errors.New("zson descriptor exists")
	ErrDescriptorInvalid = errors.New("zson descriptor out of range")
	ErrBadValue          = errors.New("malformed zson value")
	ErrBadFormat         = errors.New("malformed zson record")
)

type Reader interface {
	Read() (*Record, error)
}

type Writer interface {
	Write(*Record) error
}

type WriteCloser interface {
	Writer
	io.Closer
}

type WriteFlusher interface {
	Writer
	Flush() error
}

// Batch is an inteface to a bundle of Records.
// Batches can be shared across goroutines via reference counting and should be
// copied on modification when the reference count is greater than 1.
type Batch interface {
	Ref()
	Unref()
	Index(int) *Record
	Length() int
	Records() []*Record
	//XXX span should go in here?
	Span() nano.Span
}

func CopyWithContext(ctx context.Context, dst WriteFlusher, src Reader) error {
	var err error
	for ctx.Err() == nil {
		var rec *Record
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
