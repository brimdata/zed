package zngio

import (
	"fmt"
	"io"

	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
)

type Writer struct {
	io.Writer
	tracker *resolver.Tracker
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:  w,
		tracker: resolver.NewTracker(),
	}
}

func (w *Writer) WriteControl(b []byte) error {
	_, err := fmt.Fprintf(w.Writer, "#!%s\n", string(b))
	return err
}

func (w *Writer) Write(r *zbuf.Record) error {
	td := r.Descriptor.ID
	if !w.tracker.Seen(td) {
		_, err := fmt.Fprintf(w.Writer, "#%d:%s\n", td, r.Descriptor.Type)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w.Writer, "%d:", td)
	if err != nil {
		return nil
	}
	if err = w.writeContainer(r.Type, r.Raw); err != nil {
		return err
	}
	return w.write("\n")
}

func (w *Writer) write(s string) error {
	_, err := w.Writer.Write([]byte(s))
	return err
}

func (w *Writer) writeContainer(typ zng.Type, val []byte) error {
	if val == nil {
		w.write("-;")
		return nil
	}
	if err := w.write("["); err != nil {
		return err
	}
	childType, columns := zng.ContainedType(typ)
	if childType == nil && columns == nil {
		return zbuf.ErrSyntax
	}
	k := 0
	if len(val) > 0 {
		for it := zcode.Iter(val); !it.Done(); {
			v, container, err := it.Next()
			if err != nil {
				return err
			}
			if columns != nil {
				if k >= len(columns) {
					return &zbuf.RecordTypeError{Name: "<record>", Type: typ.String(), Err: zbuf.ErrExtraField}
				}
				childType = columns[k].Type
				k++
			}
			if container {
				if err := w.writeContainer(childType, v); err != nil {
					return err
				}
			} else {
				if err := w.writeValue(childType, v); err != nil {
					return err
				}
			}
		}
	}
	return w.write("]")
}

func (w *Writer) writeValue(typ zng.Type, zv zcode.Bytes) error {
	if zv == nil {
		return w.write("-;")
	}
	b, err := zng.Format(typ, zv)
	if err != nil {
		return err
	}
	if err := w.writeEscaped(b); err != nil {
		return err
	}
	return w.write(";")
}

func (w *Writer) escape(c byte) error {
	const hex = "0123456789abcdef"
	var b [4]byte
	b[0] = '\\'
	b[1] = 'x'
	b[2] = hex[c>>4]
	b[3] = hex[c&0xf]
	_, err := w.Writer.Write(b[:])
	return err
}

func (w *Writer) writeEscaped(val []byte) error {
	if len(val) == 0 {
		return nil
	}
	if len(val) == 1 && val[0] == '-' {
		return w.escape('-')
	}
	// We escape a bracket if it appears as the first byte of a value;
	// we otherwise don't need to escape brackets.
	if val[0] == '[' || val[0] == ']' {
		if err := w.escape(val[0]); err != nil {
			return err
		}
		val = val[1:]
	}
	off := 0
	for off < len(val) {
		c := val[off]
		switch c {
		case ';':
			if off > 0 {
				_, err := w.Writer.Write(val[:off])
				if err != nil {
					return err
				}
			}
			if err := w.escape(c); err != nil {
				return err
			}
			val = val[off+1:]
			off = 0
		default:
			off++
		}
	}
	var err error
	if len(val) > 0 {
		_, err = w.Writer.Write(val)
	}
	return err
}
