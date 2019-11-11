package zeek

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mccanne/zq/pkg/zson"
)

var ErrDescriptorChanged = errors.New("descriptor changed")

type Writer struct {
	zson.Writer
	header
	descriptor *zson.Descriptor
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{Writer: zson.Writer{w}}
}

func (z *Writer) Write(r *zson.Record) error {
	if r.Descriptor != z.descriptor {
		z.writeHeader(r)
		z.descriptor = r.Descriptor
	}
	values, err := r.Strings()
	if err != nil {
		return err
	}
	if i, ok := r.Descriptor.ColumnOfField("_path"); ok {
		// delete _path column
		values = append(values[:i], values[i+1:]...)
	}
	out := strings.Join(values, "\t") + "\n"
	_, err = z.Writer.Write([]byte(out))
	return err
}

//XXX fix this to transmit just updates
func (w *Writer) writeHeader(r *zson.Record) error {
	d := r.Descriptor
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
	if path, err := r.AccessString("_path"); err == nil && w.path != path {
		w.path = path
		s += fmt.Sprintf("#path\t%s\n", path)
	}
	if d != w.descriptor {
		s += "#fields"
		for _, col := range d.Type.Columns {
			if col.Name == "_path" {
				continue
			}
			s += fmt.Sprintf("\t%s", col.Name)
		}
		s += "\n"
		s += "#types"
		for _, col := range d.Type.Columns {
			if col.Name == "_path" {
				continue
			}
			s += fmt.Sprintf("\t%s", col.Type)
		}
		s += "\n"
	}
	_, err := w.Writer.Write([]byte(s))
	return err
}
