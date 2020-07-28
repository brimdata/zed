package zeekio

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var ErrDescriptorChanged = errors.New("descriptor changed")

type Writer struct {
	io.Writer
	header
	flattener *Flattener
	typ       *zng.TypeRecord
	precision int
	format    zng.OutFmt
}

func NewWriter(w io.Writer, flags zio.WriterFlags) *Writer {
	var format zng.OutFmt
	if flags.UTF8 {
		format = zng.OutFormatZeek
	} else {
		format = zng.OutFormatZeekAscii
	}
	return &Writer{
		Writer:    w,
		flattener: NewFlattener(resolver.NewContext()),
		precision: 6,
		format:    format,
	}
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
	values, changePrecision, err := ZeekStrings(r, w.precision, w.format)
	if err != nil {
		return err
	}
	if changePrecision {
		w.precision = 9
	}
	if i, ok := r.ColumnOfField("_path"); ok {
		// delete _path column
		values = append(values[:i], values[i+1:]...)
	}
	out := strings.Join(values, "\t") + "\n"
	_, err = w.Writer.Write([]byte(out))
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
	_, err := w.Writer.Write([]byte(s))
	return err
}

func isHighPrecision(ts nano.Ts) bool {
	_, ns := ts.Split()
	return (ns/1000)*1000 != ns
}

// This returns the zeek strings for this record.  XXX We need to not use this.
// XXX change to Pretty for output writers?... except zeek?
func ZeekStrings(r *zng.Record, precision int, fmt zng.OutFmt) ([]string, bool, error) {
	var ss []string
	it := r.ZvalIter()
	var changePrecision bool
	for _, col := range r.Type.Columns {
		val, _, err := it.Next()
		if err != nil {
			return nil, false, err
		}
		var field string
		if val == nil {
			field = "-"
		} else if precision >= 0 && col.Type == zng.TypeTime {
			ts, err := zng.DecodeTime(val)
			if err != nil {
				return nil, false, err
			}
			if precision == 6 && isHighPrecision(ts) {
				precision = 9
				changePrecision = true
			}
			field = string(ts.AppendFloat(nil, precision))
		} else {
			field = col.Type.StringOf(val, fmt, false)
		}
		ss = append(ss, field)
	}
	return ss, changePrecision, nil
}
