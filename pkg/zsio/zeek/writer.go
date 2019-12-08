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
	io.Writer
	header
	flattener  *Flattener
	descriptor *zson.Descriptor
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:    w,
		flattener: NewFlatener(),
	}
}

func (w *Writer) Write(r *zson.Record) error {
	r, err := w.flattener.Flatten(r)
	if err != nil {
		return err
	}
	path, _ := r.AccessString("_path")
	if r.Descriptor != w.descriptor || path != w.path {
		w.writeHeader(r, path)
		w.descriptor = r.Descriptor
	}
	values, err := r.ZeekStrings()
	if err != nil {
		return err
	}
	if i, ok := r.Descriptor.ColumnOfField("_path"); ok {
		// delete _path column
		values = append(values[:i], values[i+1:]...)
	}
	out := strings.Join(values, "\t") + "\n"
	_, err = w.Writer.Write([]byte(out))
	return err
}

func (w *Writer) writeHeader(r *zson.Record, path string) error {
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
	if path != w.path {
		w.path = path
		if path == "" {
			path = "-"
		}
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
