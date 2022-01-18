package textio

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zio/zeekio"
)

type Writer struct {
	writer    io.WriteCloser
	flattener *expr.Flattener
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer:    w,
		flattener: expr.NewFlattener(zed.NewContext()),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(rec *zed.Value) error {
	rec, err := w.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	var out []string
	for k, col := range zed.TypeRecordOf(rec.Type).Columns {
		var s string
		value := rec.ValueByColumn(k)
		if col.Type == zed.TypeTime {
			if value.IsNull() {
				s = "-"
			} else {
				s = zed.DecodeTime(value.Bytes).Time().Format(time.RFC3339Nano)
			}
		} else {
			s = zeekio.FormatValue(value)
		}
		out = append(out, s)
	}
	s := strings.Join(out, "\t")
	_, err = fmt.Fprintf(w.writer, "%s\n", s)
	return err
}
