package zsio

import (
	"fmt"
	"io"
	"strings"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zval"
)

type Writer struct {
	zson.Writer
	descriptors map[*zson.Descriptor]int
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		Writer: zson.Writer{w},
		descriptors: make(map[*zson.Descriptor]int),
	}
}

func (w *Writer) Write(r *zson.Record) error {
	id, ok := w.descriptors[r.Descriptor]
	if !ok {
		id = len(w.descriptors) + 1
		w.descriptors[r.Descriptor] = id
		err := w.writeDescriptor(id, r.Descriptor)
		if err != nil {
			return err
		}
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%d:", id))

	// skip the descriptor
	// XXX this should be in raw.go
	_, n := zval.Uvarint(r.Raw)
	err := zsonString(&builder, r.Descriptor.Type, r.Raw[n:])
	if err != nil {
		return err
	}
	builder.WriteByte('\n')

	_, err = w.Writer.Write([]byte(builder.String()))
	return err
}

func (w *Writer) writeDescriptor(id int, d *zson.Descriptor) error {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("#%d:record[", id))

	first := true
	for _, col := range(d.Type.Columns) {
		if first {
			first = false
		} else {
			builder.Write([]byte(","))
		}
		builder.WriteString(col.Name)
		builder.WriteByte(':')
		builder.WriteString(col.Type.String())
	}

	builder.WriteString("]\n")
	_, err := w.Writer.Write([]byte(builder.String()))
	return err
}

func zsonString(builder* strings.Builder, typ zeek.Type, val []byte) error {
	if val == nil {
		builder.WriteString("-;")
		return nil
	}
	switch t := typ.(type) {
	case *zeek.TypeSet, *zeek.TypeVector:
		builder.WriteByte('[')

		if len(val) > 0 {
			for it := zval.Iter(val); !it.Done(); {
				v, err := it.Next()
				if err != nil {
					return err
				}
				builder.WriteString(zsonEscape(string(v)))
				builder.WriteByte(';')
			}
		}
		builder.Write([]byte("];"))

	case *zeek.TypeRecord:
		builder.WriteByte('[')
		it := zval.Iter(val)
		for _, col := range(t.Columns) {
			val, err := it.Next()
			if err != nil {
				return err
			}
			err = zsonString(builder, col.Type, val)
			if err != nil {
				return err
			}
		}
		builder.WriteString("];")

	default:
		builder.WriteString(zsonEscape(string(val)))
		builder.WriteByte(';')
	}

	return nil
}

func zsonEscape(s string) string {
	if s == "-" {
		return "\\-"
	}

	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
