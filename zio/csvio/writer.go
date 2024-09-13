package csvio

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
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
	arena     *zed.Arena
	mapper    *zed.Mapper
}

type WriterOpts struct {
	Delim rune
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	encoder := csv.NewWriter(w)
	if opts.Delim != 0 {
		encoder.Comma = opts.Delim
	}
	zctx := zed.NewContext()
	return &Writer{
		writer:    w,
		encoder:   encoder,
		flattener: expr.NewFlattener(zctx),
		types:     make(map[int]struct{}),
		arena:     zed.NewArena(),
		mapper:    zed.NewMapper(zctx),
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

func (w *Writer) Write(rec zed.Value) error {
	if rec.Type().Kind() != zed.RecordKind {
		return fmt.Errorf("CSV output encountered non-record value: %s", zson.FormatValue(rec))
	}
	w.arena.Reset()
	rec, err := w.flattener.Flatten(w.arena, rec)
	if err != nil {
		return err
	}
	if w.first == nil {
		w.first = zed.TypeRecordOf(rec.Type())
		var hdr []string
		for _, f := range rec.Fields() {
			hdr = append(hdr, f.Name)
		}
		if err := w.encoder.Write(hdr); err != nil {
			return err
		}
	} else if _, ok := w.types[rec.Type().ID()]; !ok {
		if !fieldNamesEqual(w.first.Fields, rec.Fields()) {
			return ErrNotDataFrame
		}
		w.types[rec.Type().ID()] = struct{}{}
	}
	w.strings = w.strings[:0]
	fields := rec.Fields()
	for i, it := 0, rec.Bytes().Iter(); i < len(fields) && !it.Done(); i++ {
		var s string
		if zb := it.Next(); zb != nil {
			typ, err := w.mapper.Enter(fields[i].Type)
			if err != nil {
				return err
			}
			val := w.arena.New(typ, zb).Under(w.arena)
			switch id := val.Type().ID(); {
			case id == zed.IDBytes && len(val.Bytes()) == 0:
				// We want "" instead of "0x" for a zero-length value.
			case id == zed.IDString:
				s = string(val.Bytes())
			case id < zed.IDTypeComplex:
				// Avoid ZSON decoration
				s = zson.FormatPrimitive(val.Type(), val.Bytes())
				if zed.IsFloat(id) && strings.HasSuffix(s, ".") {
					s = strings.TrimSuffix(s, ".")
				}
			default:
				s = zson.FormatValue(val)
			}
		}
		w.strings = append(w.strings, s)
	}
	return w.encoder.Write(w.strings)
}

func fieldNamesEqual(a, b []zed.Field) bool {
	return slices.EqualFunc(a, b, func(a, b zed.Field) bool {
		return a.Name == b.Name
	})
}
