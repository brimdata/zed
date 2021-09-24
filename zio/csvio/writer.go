package csvio

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
)

var ErrNotDataFrame = errors.New("CSV output requires uniform records but multiple types encountered (consider 'fuse')")

type Writer struct {
	writer    io.WriteCloser
	encoder   *csv.Writer
	flattener *expr.Flattener
	first     *zed.TypeRecord
}

type WriterOpts struct {
	UTF8 bool
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer:    w,
		encoder:   csv.NewWriter(w),
		flattener: expr.NewFlattener(zed.NewContext()),
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
		var s string
		// O(n^2)
		value := rec.ValueByColumn(k)
		if !value.IsUnsetOrNil() {
			id := col.Type.ID()
			switch {
			case id == zed.IDBytes && len(value.Bytes) == 0:
				// We want "" instead of "0x" from
				// value.Type.Format.
			case zed.IsStringy(id):
				s = string(value.Bytes)

			default:
				s = value.Type.Format(value.Bytes)
				if zed.IsFloat(id) && strings.HasSuffix(s, ".") {
					s = strings.TrimSuffix(s, ".")
				}
			}
		}
		out = append(out, s)
	}
	return w.encoder.Write(out)
}
