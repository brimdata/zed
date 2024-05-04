package zeekio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
)

type Writer struct {
	writer io.WriteCloser

	buf bytes.Buffer
	header
	flattener *expr.Flattener
	typ       *zed.TypeRecord
	zctx      *zed.Context
	arena     *zed.Arena
	mapper    *zed.Mapper
}

func NewWriter(w io.WriteCloser) *Writer {
	zctx := zed.NewContext()
	return &Writer{
		writer:    w,
		flattener: expr.NewFlattener(zctx),
		zctx:      zctx,
		arena:     zed.NewArena(),
		mapper:    zed.NewMapper(zctx),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(r zed.Value) error {
	w.arena.Reset()
	r, err := w.flattener.Flatten(w.arena, r)
	if err != nil {
		return err
	}
	path := r.Deref(w.arena, "_path").AsString()
	if r.Type() != w.typ || path != w.Path {
		if err := w.writeHeader(r, path); err != nil {
			return err
		}
		w.typ = zed.TypeRecordOf(r.Type())
	}
	w.buf.Reset()
	var needSeparator bool
	it := r.Bytes().Iter()
	for _, f := range zed.TypeRecordOf(r.Type()).Fields {
		bytes := it.Next()
		if f.Name == "_path" {
			continue
		}
		if needSeparator {
			w.buf.WriteByte('\t')
		}
		needSeparator = true
		typ, err := w.mapper.Enter(f.Type)
		if err != nil {
			return err
		}
		w.buf.WriteString(FormatValue(w.arena, w.arena.New(typ, bytes)))
	}
	w.buf.WriteByte('\n')
	_, err = w.writer.Write(w.buf.Bytes())
	return err
}

func (w *Writer) writeHeader(r zed.Value, path string) error {
	d := r.Type()
	var s string
	if w.separator != "\\x90" {
		w.separator = "\\x90"
		s += "#separator \\x09\n"
	}
	if w.setSeparator != "," {
		w.setSeparator = ","
		s += "#set_separator\t,\n"
	}
	if w.emptyField != "(empty)" {
		w.emptyField = "(empty)"
		s += "#empty_field\t(empty)\n"
	}
	if w.unsetField != "-" {
		w.unsetField = "-"
		s += "#unset_field\t-\n"
	}
	if path != w.Path {
		w.Path = path
		if path == "" {
			path = "-"
		}
		s += fmt.Sprintf("#path\t%s\n", path)
	}
	if d != w.typ {
		s += "#fields"
		for _, f := range zed.TypeRecordOf(d).Fields {
			if f.Name == "_path" {
				continue
			}
			s += fmt.Sprintf("\t%s", f.Name)
		}
		s += "\n"
		s += "#types"
		for _, f := range zed.TypeRecordOf(d).Fields {
			if f.Name == "_path" {
				continue
			}
			t, err := zngTypeToZeek(f.Type)
			if err != nil {
				return err
			}
			s += fmt.Sprintf("\t%s", t)
		}
		s += "\n"
	}
	_, err := w.writer.Write([]byte(s))
	return err
}
