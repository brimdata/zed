package textio

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zio/tzngio"
)

type Writer struct {
	writer    io.WriteCloser
	flattener *expr.Flattener
	format    tzngio.OutFmt
}

func NewWriter(w io.WriteCloser, utf8 bool) *Writer {
	format := tzngio.OutFormatZeekAscii
	if utf8 {
		format = tzngio.OutFormatZeek
	}
	return &Writer{
		writer:    w,
		flattener: expr.NewFlattener(zed.NewContext()),
		format:    format,
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
			if value.IsUnsetOrNil() {
				s = "-"
			} else {
				ts, err := zed.DecodeTime(value.Bytes)
				if err != nil {
					return err
				}
				s = ts.Time().UTC().Format(time.RFC3339Nano)
			}
		} else {
			s = tzngio.FormatValue(value, w.format)
		}
		out = append(out, s)
	}
	s := strings.Join(out, "\t")
	_, err = fmt.Fprintf(w.writer, "%s\n", s)
	return err
}
