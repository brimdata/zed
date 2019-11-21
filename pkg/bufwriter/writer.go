// Package bufwriter provides a wrapper for a io.WriteCloser that uses
// buffered output via a bufio.Writer and calls Flush on close.
// Not clear why bufio.Writer doesn't do this.
package bufwriter

import (
	"bufio"
	"io"
)

type Writer struct {
	closer io.Closer
	*bufio.Writer
}

func New(w io.WriteCloser) *Writer {
	return &Writer{
		closer: w,
		Writer: bufio.NewWriter(w),
	}
}

func (w *Writer) Close() error {
	if err := w.Writer.Flush(); err != nil {
		return err
	}
	return w.closer.Close()
}
