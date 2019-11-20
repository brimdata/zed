// Package bufwriter provides a wrapper for io.WriteCloser that uses
// buffered output via a bufio.Writer and calls Flush on close.
// Not clear why bufio.Writer doesn't do this.
package bufwriter

import (
	"bufio"
	"io"
)

type Writer struct {
	io.WriteCloser
	writer *bufio.Writer
}

func New(w io.WriteCloser) io.WriteCloser {
	return &Writer{
		WriteCloser: w,
		writer:      bufio.NewWriter(w),
	}
}

func (w *Writer) Close() error {
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.WriteCloser.Close()
}
