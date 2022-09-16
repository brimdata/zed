package zio

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"golang.org/x/exp/slices"
)

func Extension(format string) string {
	switch format {
	case "zeek":
		return ".log"
	case "json":
		return ".json"
	case "zjson":
		return ".ndjson"
	case "text":
		return ".txt"
	case "table":
		return ".tbl"
	case "zng":
		return ".zng"
	case "zson":
		return ".zson"
	case "csv":
		return ".csv"
	case "zst":
		return ".zst"
	case "parquet":
		return ".parquet"
	default:
		return ""
	}
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// NopCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

// Reader wraps the Read method.
//
// Read returns the next record and a nil error, a nil record and the next
// error, or a nil record and nil error to indicate that no records remain.
//
// Read never returns a non-nil record and non-nil error together, and it never
// returns io.EOF.
type Reader interface {
	Read() (*zed.Value, error)
}

type Writer interface {
	Write(*zed.Value) error
}

type ReadCloser interface {
	Reader
	io.Closer
}

type WriteCloser interface {
	Writer
	io.Closer
}

func NewReadCloser(r Reader, c io.Closer) ReadCloser {
	return extReadCloser{r, c}
}

type extReadCloser struct {
	Reader
	io.Closer
}

func NopReadCloser(r Reader) ReadCloser {
	return nopReadCloser{r}
}

type nopReadCloser struct {
	Reader
}

func (nopReadCloser) Close() error { return nil }

// ConcatReader returns a Reader that is the logical concatenation of readers,
// which are read sequentially.  Its Read methed returns any non-nil error
// returned by a reader and returns end of stream after all readers have
// returned end of stream.
func ConcatReader(readers ...Reader) Reader {
	if len(readers) == 1 {
		return readers[0]
	}
	return &concatReader{slices.Clone(readers)}
}

type concatReader struct {
	readers []Reader
}

func (c *concatReader) Read() (*zed.Value, error) {
	for len(c.readers) > 0 {
		rec, err := c.readers[0].Read()
		if rec != nil || err != nil {
			return rec, err
		}
		c.readers = c.readers[1:]
	}
	return nil, nil
}

func MultiWriter(writers ...Writer) Writer {
	return &multiWriter{slices.Clone(writers)}
}

type multiWriter struct {
	writers []Writer
}

func (m *multiWriter) Write(rec *zed.Value) error {
	for _, w := range m.writers {
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

// Copy copies src to dst a la io.Copy.
func Copy(dst Writer, src Reader) error {
	return CopyWithContext(context.Background(), dst, src)
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
