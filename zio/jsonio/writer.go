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
	recs   []interface{}
	size   int
}

type describe struct {
	Kind  string     `json:"kind"`
	Value *zed.Value `json:"value"`
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	return &Writer{
		writer: w,
		opts:   opts,
	}
}

func (w *Writer) Close() error {
	var body interface{} = w.recs
	if len(w.recs) == 1 && !w.opts.ForceArray {
		body = w.recs[0]
	}
	err := json.NewEncoder(w.writer).Encode(body)
	if closeErr := w.writer.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (w *Writer) Write(rec *zed.Value) error {
	if w.size > MaxWriteBuffer {
		return fmt.Errorf("JSON output buffer size exceeded: %d", w.size)
	}
	if alias, ok := rec.Type.(*zed.TypeAlias); ok {
		w.recs = append(w.recs, &describe{alias.Name, rec.Keep()})
	} else {
		w.recs = append(w.recs, rec.Keep())
	}
	w.size += len(rec.Bytes)
	return nil
}
