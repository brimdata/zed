package zbuf

import (
	"context"
	"io"

	"github.com/brimsec/zq/zng"
)

// Reader wraps the Read method.
//
// Read returns the next record and a nil error, a nil record and the next
// error, or a nil record and nil error to indicate that no records remain.
//
// Read never returns a non-nil record and non-nil error together, and it never
// returns io.EOF.
type Reader interface {
	Read() (*zng.Record, error)
}

type ReadCloser interface {
	Reader
	io.Closer
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

type nopReadCloser struct {
	Reader
}

func (nopReadCloser) Close() error { return nil }

func NopReadCloser(r Reader) ReadCloser {
	return nopReadCloser{r}
}

type extReadCloser struct {
	Reader
	io.Closer
}

func NewReadCloser(r Reader, c io.Closer) ReadCloser {
	return extReadCloser{r, c}
}

func CopyWithContext(ctx context.Context, dst Writer, src Reader) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		rec, err := src.Read()
		if err != nil || rec == nil {
			return err
		}
		if err := dst.Write(rec); err != nil {
			return err
		}
	}
}

// Copy copies src to dst a la io.Copy.
func Copy(dst Writer, src Reader) error {
	return CopyWithContext(context.Background(), dst, src)
}

func MultiWriter(writers ...Writer) Writer {
	w := make([]Writer, len(writers))
	copy(w, writers)
	return &multiWriter{w}
}

type multiWriter struct {
	writers []Writer
}

func (m *multiWriter) Write(rec *zng.Record) error {
	for _, w := range m.writers {
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}
