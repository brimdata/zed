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
		flattener:  expr.NewFlattener(zed.NewContext()),
		format:     format,
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
	it := rec.Bytes.Iter()
	for _, col := range zed.TypeRecordOf(rec.Type).Columns {
		var s, v string
		bytes, _, err := it.Next()
		if err != nil {
			return err
		}
		if col.Type == zed.TypeTime {
			if bytes == nil {
				v = "-"
			} else {
				ts, err := zed.DecodeTime(bytes)
				if err != nil {
					return err
				}
				v = ts.Time().UTC().Format(time.RFC3339Nano)
			}
		} else {
			v = tzngio.FormatValue(zed.Value{col.Type, bytes}, w.format)
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
