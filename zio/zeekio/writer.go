package zeekio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
)

type Writer struct {
	writer io.WriteCloser

	buf bytes.Buffer
	header
	flattener *expr.Flattener
	typ       *zed.TypeRecord
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer:    w,
		flattener: expr.NewFlattener(zed.NewContext()),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(r *zed.Value) error {
	r, err := w.flattener.Flatten(r)
	if err != nil {
		return err
	}
	path := r.Deref("_path").AsString()
	if r.Type != w.typ || path != w.Path {
		if err := w.writeHeader(r, path); err != nil {
			return err
		}
		w.typ = zed.TypeRecordOf(r.Type)
	}
	w.buf.Reset()
	var needSeparator bool
	it := r.Bytes.Iter()
	for _, col := range zed.TypeRecordOf(r.Type).Columns {
		bytes := it.Next()
		if col.Name == "_path" {
			continue
		}
		if needSeparator {
			w.buf.WriteByte('\t')
		}
		needSeparator = true
		w.buf.WriteString(FormatValue(*zed.NewValue(col.Type, bytes)))
	}
	w.buf.WriteByte('\n')
	_, err = w.writer.Write(w.buf.Bytes())
	return err
}

func (w *Writer) writeHeader(r *zed.Value, path string) error {
	d := r.Type
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
		for _, col := range zed.TypeRecordOf(d).Columns {
			if col.Name == "_path" {
				continue
			}
			s += fmt.Sprintf("\t%s", col.Name)
		}
		s += "\n"
		s += "#types"
		for _, col := range zed.TypeRecordOf(d).Columns {
			if col.Name == "_path" {
				continue
			}
			t, err := zngTypeToZeek(col.Type)
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
