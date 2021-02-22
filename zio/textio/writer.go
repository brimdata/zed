package textio

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
)

type Writer struct {
	WriterOpts
	EpochDates bool
	writer     io.WriteCloser
	flattener  *flattener.Flattener
	format     tzngio.OutFmt
}

type WriterOpts struct {
	ShowTypes  bool
	ShowFields bool
}

func NewWriter(w io.WriteCloser, utf8 bool, opts WriterOpts, dates bool) *Writer {
	format := tzngio.OutFormatZeekAscii
	if utf8 {
		format = tzngio.OutFormatZeek
	}
	return &Writer{
		WriterOpts: opts,
		EpochDates: dates,
		writer:     w,
		flattener:  flattener.New(resolver.NewContext()),
		format:     format,
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(rec *zng.Record) error {
	rec, err := w.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	var out []string
	if w.ShowFields || w.ShowTypes || !w.EpochDates {
		for k, col := range rec.Type.Columns {
			var s, v string
			value := rec.Value(k)
			if !w.EpochDates && col.Type == zng.TypeTime {
				if value.IsUnsetOrNil() {
					v = "-"
				} else {
					ts, err := zng.DecodeTime(value.Bytes)
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
	} else {
		var err error
		out, err = zeekio.ZeekStrings(rec, w.format)
		if err != nil {
			return err
		}
	}
	s := strings.Join(out, "\t")
	_, err = fmt.Fprintf(w.writer, "%s\n", s)
	return err
}
