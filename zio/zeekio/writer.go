package zeekio

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
)

var ErrDescriptorChanged = errors.New("descriptor changed")

type Writer struct {
	writer io.WriteCloser
	header
	buffer    bytes.Buffer
	flattener *flattener.Flattener
	typ       *zng.TypeRecord
	format    zng.OutFmt
}

func NewWriter(w io.WriteCloser, utf8 bool) *Writer {
	var format zng.OutFmt
	if utf8 {
		format = zng.OutFormatZeek
	} else {
		format = zng.OutFormatZeekAscii
	}
	return &Writer{
		writer:    w,
		flattener: flattener.New(resolver.NewContext()),
		format:    format,
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
	path, _ := r.AccessString("_path")
	if r.Type != w.typ || path != w.Path {
		if err := w.writeHeader(r, path); err != nil {
			return err
		}
		w.typ = r.Type
	}
	w.buffer.Reset()
	var sep string
	for k, col := range r.Type.Columns {
		if col.Name == "_path" {
			continue
		}
		w.buffer.WriteString(sep)
		sep = "\t"
		value := r.Value(k)
		var s string
		if value.IsUnsetOrNil() {
			s = "-"
		} else if col.Type == zng.TypeTime {
			ts, err := zng.DecodeTime(value.Bytes)
			if err != nil {
				return err
			}
			precision := 6
			if isHighPrecision(ts) {
				precision = 9
			}
			s = string(ts.AppendFloat(nil, precision))
		} else {
			s = value.Format(w.format)
		}
		w.buffer.WriteString(s)
	}
	w.buffer.WriteByte('\n')
	_, err = w.writer.Write(w.buffer.Bytes())
	return err
}

func (w *Writer) writeHeader(r *zng.Record, path string) error {
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
		for _, col := range d.Columns {
			if col.Name == "_path" {
				continue
			}
			s += fmt.Sprintf("\t%s", col.Name)
		}
		s += "\n"
		s += "#types"
		for _, col := range d.Columns {
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

func isHighPrecision(ts nano.Ts) bool {
	_, ns := ts.Split()
	return (ns/1000)*1000 != ns
}
