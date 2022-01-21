package jsonio

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/brimdata/zed"
)

const MaxWriteBuffer = 25 * 1024 * 1024

type WriterOpts struct {
	ForceArray bool
}

type Writer struct {
	writer io.WriteCloser
	opts   WriterOpts
	vals   []interface{}
	size   int
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	return &Writer{
		writer: w,
		opts:   opts,
	}
}

func (w *Writer) Close() error {
	body := interface{}(w.vals)
	if len(w.vals) == 1 && !w.opts.ForceArray {
		body = w.vals[0]
	}
	err := json.NewEncoder(w.writer).Encode(body)
	if closeErr := w.writer.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (w *Writer) Write(val *zed.Value) error {
	if w.size > MaxWriteBuffer {
		return fmt.Errorf("JSON output buffer size exceeded: %d", w.size)
	}
	w.vals = append(w.vals, Marshal(val))
	w.size += len(val.Bytes)
	return nil
}
