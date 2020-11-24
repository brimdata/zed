package textio

import (
	"bytes"
	"io"
	"time"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
)

type Writer struct {
	WriterOpts
	EpochDates bool
	writer     io.WriteCloser
	buffer     bytes.Buffer
	flattener  *flattener.Flattener
	format     zng.OutFmt
}

type WriterOpts struct {
	ShowTypes  bool
	ShowFields bool
}

func NewWriter(w io.WriteCloser, utf8 bool, opts WriterOpts, dates bool) *Writer {
	format := zng.OutFormatZeekAscii
	if utf8 {
		format = zng.OutFormatZeek
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

func (w *Writer) Write(r *zng.Record) error {
	r, err := w.flattener.Flatten(r)
	if err != nil {
		return err
	}
	w.buffer.Reset()
	for k, col := range r.Type.Columns {
		if k > 0 {
			w.buffer.WriteByte('\t')
		}
		if w.ShowFields {
			w.buffer.WriteString(col.Name)
			w.buffer.WriteByte(':')
		}
		if w.ShowTypes {
			w.buffer.WriteString(col.Type.String())
			w.buffer.WriteByte(':')
		}
		value := r.Value(k)
		var s string
		if !w.EpochDates && col.Type == zng.TypeTime {
			if value.IsUnsetOrNil() {
				s = "-"
			} else {
				ts, err := zng.DecodeTime(value.Bytes)
				if err != nil {
					return err
				}
				s = ts.Time().UTC().Format(time.RFC3339Nano)
			}
		} else {
			s = value.Format(w.format)
		}
		w.buffer.WriteString(s)
	}
	w.buffer.WriteByte('\n')
	_, err = w.writer.Write(w.buffer.Bytes())
	return err
}
