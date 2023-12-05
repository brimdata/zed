package csvio

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
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
	types     map[int]struct{}
	first     *zed.TypeRecord
	strings   []string
}

type WriterOpts struct {
	Delim rune
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	encoder := csv.NewWriter(w)
	if opts.Delim != 0 {
		encoder.Comma = opts.Delim
	}
	return &Writer{
		writer:    w,
		encoder:   encoder,
		flattener: expr.NewFlattener(zed.NewContext()),
		types:     make(map[int]struct{}),
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
		return fmt.Errorf("CSV output encountered non-record value: %s", zson.FormatValue(rec))
	}
	rec, err := w.flattener.Flatten(rec)
	if err != nil {
		return err
	}
	if w.first == nil {
		w.first = zed.TypeRecordOf(rec.Type)
		var hdr []string
		for _, f := range rec.Fields() {
			hdr = append(hdr, f.Name)
		}
		if err := w.encoder.Write(hdr); err != nil {
			return err
		}
	} else if _, ok := w.types[rec.Type.ID()]; !ok {
		if !fieldNamesEqual(w.first.Fields, rec.Fields()) {
			return ErrNotDataFrame
		}
		w.types[rec.Type.ID()] = struct{}{}
	}
	w.strings = w.strings[:0]
	fields := rec.Fields()
	for i, it := 0, rec.Bytes().Iter(); i < len(fields) && !it.Done(); i++ {
		var s string
		if zb := it.Next(); zb != nil {
			val := zed.NewValue(fields[i].Type, zb)
			val = val.Under(val)
			switch id := val.Type.ID(); {
			case id == zed.IDBytes && len(val.Bytes()) == 0:
				// We want "" instead of "0x" for a zero-length value.
			case id == zed.IDString:
				s = string(val.Bytes())
			default:
				s = formatValue(val.Type, val.Bytes())
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
		return zson.FormatPrimitive(zed.TypeUnder(typ), bytes)
	}
	return zson.FormatValue(zed.NewValue(typ, bytes))
}

func fieldNamesEqual(a, b []zed.Field) bool {
	return slices.EqualFunc(a, b, func(a, b zed.Field) bool {
		return a.Name == b.Name
	})
}
