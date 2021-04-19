package jsonio

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/brimdata/zed/zng"
)

const MaxWriteBuffer = 25 * 1024 * 1024

type Writer struct {
	writer io.WriteCloser
	recs   []interface{}
	size   int
}

type describe struct {
	Kind  string      `json:"kind"`
	Value *zng.Record `json:"value"`
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer: w,
	}
}

func (w *Writer) Close() error {
	err := json.NewEncoder(w.writer).Encode(w.recs)
	if closeErr := w.writer.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (w *Writer) Write(rec *zng.Record) error {
	if w.size > MaxWriteBuffer {
		return fmt.Errorf("JSON output buffer size exceeded: %d", w.size)
	}
	if alias, ok := rec.Type.(*zng.TypeAlias); ok {
		w.recs = append(w.recs, &describe{alias.Name, rec.Keep()})
	} else {
		w.recs = append(w.recs, rec.Keep())
	}
	w.size += len(rec.Bytes)
	return nil
}
