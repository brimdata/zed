package textio

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
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

func (w *Writer) Write(val *zed.Value) error {
	if _, ok := zed.TypeUnder(val.Type).(*zed.TypeRecord); ok {
		return w.writeRecord(val)
	}
	_, err := fmt.Fprintln(w.writer, zeekio.FormatValue(*val))
	return err
}

func (w *Writer) writeRecord(rec *zed.Value) error {
	rec, err := w.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	var out []string
	for k, col := range zed.TypeRecordOf(rec.Type).Columns {
		var s string
		value := rec.DerefByColumn(k).MissingAsNull()
		if col.Type == zed.TypeTime {
			if value.IsNull() {
				s = "-"
			} else {
				s = zed.DecodeTime(value.Bytes).Time().Format(time.RFC3339Nano)
			}
		} else {
			s = zeekio.FormatValue(*value)
		}
		out = append(out, s)
	}
	s := strings.Join(out, "\t")
	_, err = fmt.Fprintln(w.writer, s)
	return err
}
