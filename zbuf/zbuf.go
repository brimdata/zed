package zbuf

import (
	"context"
	"io"

	"github.com/brimdata/zq/zng"
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

func CloseReaders(readers []Reader) error {
	var err error
	for _, reader := range readers {
		if closer, ok := reader.(io.Closer); ok {
			if e := closer.Close(); err == nil {
				err = e
			}
		}
	}
	return err
}

func ReadAll(r Reader) (arr Array, err error) {
	if err := Copy(&arr, r); err != nil {
		return nil, err
	}
	return
}
