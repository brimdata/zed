package csvio

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

var ErrNotDataFrame = errors.New("CSV output requires uniform records but multiple types encountered (consider 'fuse')")

type Writer struct {
	writer    io.WriteCloser
	encoder   *csv.Writer
	flattener *expr.Flattener
	first     *zed.TypeRecord
	strings   []string
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

func (w *Writer) Write(rec *zed.Value) error {
	if rec.Type.Kind() != zed.RecordKind {
		return fmt.Errorf("CSV output encountered non-record value: %s", zson.MustFormatValue(*rec))
	}
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
	w.strings = w.strings[:0]
	cols := rec.Columns()
	for i, it := 0, rec.Bytes.Iter(); i < len(cols) && !it.Done(); i++ {
		var s string
		if zb := it.Next(); zb != nil {
			typ := cols[i].Type
			id := typ.ID()
			switch {
			case id == zed.IDBytes && len(zb) == 0:
				// We want "" instead of "0x" from typ.Format.
			case id == zed.IDString:
				s = string(zb)
			default:
				s = formatValue(typ, zb)
				if zed.IsFloat(id) && strings.HasSuffix(s, ".") {
					s = strings.TrimSuffix(s, ".")
				}
			}
		}
		w.strings = append(w.strings, s)
	}
	return w.encoder.Write(w.strings)
}

func formatValue(typ zed.Type, bytes zcode.Bytes) string {
	// Avoid ZSON decoration.
	if typ.ID() < zed.IDTypeComplex {
		return zson.FormatPrimitive(typ, bytes)
	}
	return zson.String(zed.Value{typ, bytes})
}
