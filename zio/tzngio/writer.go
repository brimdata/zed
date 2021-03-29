package tzngio

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

type Writer struct {
	writer io.WriteCloser
	// tracker keeps track of a mapping from internal ZNG type IDs for each
	// new record encountered (i.e., which triggers a typedef) so that we
	// generate the output in canonical form whereby the typedefs in the
	// stream are numbered sequentially from 0.
	tracker map[int]string
	// aliases keeps track of whether an alias has been written to the stream
	// on not.
	aliases map[int]struct{}
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer:  w,
		tracker: make(map[int]string),
		aliases: make(map[int]struct{}),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) WriteControl(b []byte) error {
	_, err := fmt.Fprintf(w.writer, "#!%s\n", string(b))
	return err
}

func (w *Writer) Write(r *zng.Record) error {
	inId := r.Type.ID()
	name, ok := w.tracker[inId]
	if !ok {
		if err := w.writeAliases(r.Type); err != nil {
			return err
		}
		typ := r.Type
		var op string
		if alias, ok := typ.(*zng.TypeAlias); ok {
			name = alias.Name
			op = "="
		} else {
			id := len(w.tracker)
			name = strconv.Itoa(id)
			op = ":"
		}
		w.tracker[inId] = name
		_, err := fmt.Fprintf(w.writer, "#%s%s%s\n", name, op, TypeString(r.Type))
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w.writer, "%s:", name)
	if err != nil {
		return nil
	}
	// XXX these write* methods are redundant with the StringOf methods on
	// zng type.  We should just call StringOf on r.Type here and get rid of
	// all these write* methods and make sure there is consistency between this
	// logic and the logic in the StringOfs.  See issue #1417.
	if err := w.writeContainer(r.Value); err != nil {
		return err
	}
	return w.write("\n")
}

func (w *Writer) writeAliases(typ zng.Type) error {
	aliases := zng.AliasTypes(typ)
	for _, alias := range aliases {
		id := alias.AliasID()
		if _, ok := w.aliases[id]; !ok {
			w.aliases[id] = struct{}{}
			_, err := fmt.Fprintf(w.writer, "#%s=%s\n", alias.Name, TypeString(alias.Type))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Writer) write(s string) error {
	_, err := w.writer.Write([]byte(s))
	return err
}

func (w *Writer) writeUnion(parent zng.Value) error {
	utyp := zng.AliasOf(parent.Type).(*zng.TypeUnion)
	inner, index, v, err := utyp.SplitZng(parent.Bytes)
	if err != nil {
		return err
	}
	s := strconv.FormatInt(index, 10) + ":"
	if err := w.write(s); err != nil {
		return err
	}

	value := zng.Value{inner, v}
	if zng.IsContainerType(zng.AliasOf(inner)) {
		if err := w.writeContainer(value); err != nil {
			return err
		}
	} else {
		if err := w.writeValue(value); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) writeContainer(parent zng.Value) error {
	if parent.IsUnsetOrNil() {
		w.write("-;")
		return nil
	}
	realType := zng.AliasOf(parent.Type)
	if _, ok := realType.(*zng.TypeUnion); ok {
		return w.writeUnion(parent)
	}
	if typ, ok := realType.(*zng.TypeMap); ok {
		s := StringOf(zng.Value{typ, parent.Bytes}, OutFormatZNG, true)
		return w.write(s)
	}
	if err := w.write("["); err != nil {
		return err
	}
	childType, columns := zng.ContainedType(realType)
	if childType == nil && columns == nil {
		return ErrSyntax
	}
	k := 0
	if len(parent.Bytes) > 0 {
		for it := parent.Bytes.Iter(); !it.Done(); {
			v, container, err := it.Next()
			if err != nil {
				return err
			}
			if columns != nil {
				if k >= len(columns) {
					return &zng.RecordTypeError{Name: "<record>", Type: parent.Type.ZSON(), Err: zng.ErrExtraField}
				}
				childType = columns[k].Type
				k++
			}
			value := zng.Value{childType, v}
			if container {
				if err := w.writeContainer(value); err != nil {
					return err
				}
			} else {
				if err := w.writeValue(value); err != nil {
					return err
				}
			}
		}
	}
	return w.write("]")
}

func (w *Writer) writeValue(v zng.Value) error {
	if v.IsUnsetOrNil() {
		return w.write("-;")
	}
	if err := w.write(FormatValue(v, OutFormatZNG)); err != nil {
		return err
	}
	return w.write(";")
}

func WriteString(w zbuf.Writer, s string) error {
	r := NewReader(strings.NewReader(s), resolver.NewContext())
	return zbuf.Copy(w, r)
}
