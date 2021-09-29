package textio

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zson"
)

type Writer struct {
	WriterOpts
	writer    io.WriteCloser
	flattener *expr.Flattener
	format    tzngio.OutFmt
}

type WriterOpts struct {
	ShowTypes  bool
	ShowFields bool
}

func NewWriter(w io.WriteCloser, utf8 bool, opts WriterOpts) *Writer {
	format := tzngio.OutFormatZeekAscii
	if utf8 {
		format = tzngio.OutFormatZeek
	}
	return &Writer{
		WriterOpts: opts,
		writer:     w,
		flattener:  expr.NewFlattener(zson.NewContext()),
		format:     format,
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(rec *zed.Record) error {
	rec, err := w.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	var out []string
	for k, col := range zed.TypeRecordOf(rec.Type).Columns {
		var s, v string
		value := rec.ValueByColumn(k)
		if col.Type == zed.TypeTime {
			if value.IsUnsetOrNil() {
				v = "-"
			} else {
				ts, err := zed.DecodeTime(value.Bytes)
				if err != nil {
					return err
				}
				v = ts.Time().UTC().Format(time.RFC3339Nano)
			}
		} else {
			v = tzngio.FormatValue(value, w.format)
		}
		if w.ShowFields {
			s = col.Name + ":"
		}
		if w.ShowTypes {
			s = s + tzngio.TypeString(col.Type) + ":"
		}
		out = append(out, s+v)
	}
	s := strings.Join(out, "\t")
	_, err = fmt.Fprintf(w.writer, "%s\n", s)
	return err
}
