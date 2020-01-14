package zbuf

import (
	"context"
	"errors"
	"io"

	"github.com/mccanne/zq/pkg/nano"
)

var (
	ErrDescriptorExists  = errors.New("zng descriptor exists")
	ErrDescriptorInvalid = errors.New("zng descriptor out of range")
	ErrBadValue          = errors.New("malformed zng value")
	ErrBadFormat         = errors.New("malformed zng record")
	ErrTypeMismatch      = errors.New("type/value mismatch")
	ErrNoSuchField       = errors.New("no such field in zng record")
	ErrNoSuchColumn      = errors.New("no such column in zng record")
	ErrCorruptTd         = errors.New("corrupt type descriptor")
	ErrCorruptColumns    = errors.New("wrong number of columns in zng record value")
)

type RecordTypeError struct {
	Name string
	Type string
	Err  error
}

func (r *RecordTypeError) Error() string { return r.Name + " (" + r.Type + "): " + r.Err.Error() }
func (r *RecordTypeError) Unwrap() error { return r.Err }

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

type nopFlusher struct {
	Writer
}

func (nopFlusher) Flush() error { return nil }

// NopFlusher returns a WriteFlusher with a no-op Flush method wrapping
// the provided Writer w.
func NopFlusher(w Writer) WriteFlusher {
	return nopFlusher{w}
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
