package zeekio

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
)

var ErrDescriptorChanged = errors.New("descriptor changed")

type Writer struct {
	writer io.WriteCloser
	header
	flattener *flattener.Flattener
	typ       *zng.TypeRecord
	format    tzngio.OutFmt
}

func NewWriter(w io.WriteCloser, utf8 bool) *Writer {
	var format tzngio.OutFmt
	if utf8 {
		format = tzngio.OutFormatZeek
	} else {
		format = tzngio.OutFormatZeekAscii
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
		w.typ = zng.TypeRecordOf(r.Type)
	}
	values, err := ZeekStrings(r, w.format)
	if err != nil {
		return err
	}
	if i, ok := r.ColumnOfField("_path"); ok {
		// delete _path column
		values = append(values[:i], values[i+1:]...)
	}
	out := strings.Join(values, "\t") + "\n"
	_, err = w.writer.Write([]byte(out))
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
		for _, col := range zng.TypeRecordOf(d).Columns {
			if col.Name == "_path" {
				continue
			}
			s += fmt.Sprintf("\t%s", col.Name)
		}
		s += "\n"
		s += "#types"
		for _, col := range zng.TypeRecordOf(d).Columns {
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

// This returns the zeek strings for this record.  XXX We need to not use this.
// XXX change to Pretty for output writers?... except zeek?
func ZeekStrings(r *zng.Record, fmt tzngio.OutFmt) ([]string, error) {
	var ss []string
	it := r.ZvalIter()
	for _, col := range zng.TypeRecordOf(r.Type).Columns {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		var field string
		if val == nil {
			field = "-"
		} else if col.Type == zng.TypeTime {
			ts, err := zng.DecodeTime(val)
			if err != nil {
				return nil, err
			}
			precision := 6
			if isHighPrecision(ts) {
				precision = 9
			}
			field = string(ts.AppendFloat(nil, precision))
		} else {
			field = tzngio.StringOf(zng.Value{col.Type, val}, fmt, false)
		}
		ss = append(ss, field)
	}
	return ss, nil
}
