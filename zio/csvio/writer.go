package csvio

import (
	"encoding/csv"
	"errors"
	"io"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zio/tzngio"
	"github.com/brimdata/zed/zson"
)

var ErrNotDataFrame = errors.New("CSV output requires uniform records but multiple types encountered (consider 'fuse')")

type Writer struct {
	writer    io.WriteCloser
	encoder   *csv.Writer
	flattener *expr.Flattener
	format    tzngio.OutFmt
	first     *zed.TypeRecord
}

type WriterOpts struct {
	UTF8 bool
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	format := tzngio.OutFormatZeekAscii
	if opts.UTF8 {
		format = tzngio.OutFormatZeek
	}
	return &Writer{
		writer:    w,
		encoder:   csv.NewWriter(w),
		flattener: expr.NewFlattener(zson.NewContext()),
		format:    format,
	}
}

func (w *Writer) Close() error {
	w.encoder.Flush()
	return w.writer.Close()
}

func (w *Writer) Flush() error {
	w.encoder.Flush()
	return w.encoder.Error()
}

func (w *Writer) Write(rec *zed.Record) error {
	rec, err := w.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	if w.first == nil {
		w.first = zed.TypeRecordOf(rec.Type)
		var hdr []string
		for _, col := range rec.Columns() {
			hdr = append(hdr, col.Name)
		}
		if err := w.encoder.Write(hdr); err != nil {
			return err
		}
	} else if rec.Type != w.first {
		return ErrNotDataFrame
	}
	var out []string
	for k, col := range rec.Columns() {
		var v string
		// O(n^2)
		value := rec.ValueByColumn(k)
		if !value.IsUnsetOrNil() {
			switch col.Type.ID() {
			case zed.IDTime:
				ts, err := zed.DecodeTime(value.Bytes)
				if err != nil {
					return err
				}
				v = ts.Time().UTC().Format(time.RFC3339Nano)
			case zed.IDString, zed.IDBstring, zed.IDType, zed.IDError:
				v = string(value.Bytes)
			default:
				v = tzngio.FormatValue(value, w.format)
			}
		}
		out = append(out, v)
	}
	return w.encoder.Write(out)
}
