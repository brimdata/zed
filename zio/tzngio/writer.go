package tzngio

import (
	"fmt"
	"io"
	"strconv"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type Writer struct {
	io.Writer
	// tracker keeps track of a mapping from internal ZNG type IDs for each
	// new record encountered (i.e., which triggers a typedef) so that we
	// generate the output in canonical form whereby the typedefs in the
	// stream are numbered sequentially from 0.
	tracker map[int]int
	// aliases keeps track of whether an alias has been written to the stream
	// on not.
	aliases map[int]struct{}
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Writer:  w,
		tracker: make(map[int]int),
		aliases: make(map[int]struct{}),
	}
}

func (w *Writer) WriteControl(b []byte) error {
	_, err := fmt.Fprintf(w.Writer, "#!%s\n", string(b))
	return err
}

func (w *Writer) Write(r *zng.Record) error {
	inId := r.Type.ID()
	outId, ok := w.tracker[inId]
	if !ok {
		if err := w.writeAliases(r); err != nil {
			return err
		}
		outId = len(w.tracker)
		w.tracker[inId] = outId
		_, err := fmt.Fprintf(w.Writer, "#%d:%s\n", outId, r.Type)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w.Writer, "%d:", outId)
	if err != nil {
		return nil
	}
	if err := w.writeContainer(zng.Value{Type: r.Type, Bytes: r.Raw}); err != nil {
		return err
	}
	return w.write("\n")
}

func (w *Writer) writeAliases(r *zng.Record) error {
	aliases := zng.AliasTypes(r.Type)
	for _, alias := range aliases {
		id := alias.AliasID()
		if _, ok := w.aliases[id]; !ok {
			w.aliases[id] = struct{}{}
			_, err := fmt.Fprintf(w.Writer, "#%s=%s\n", alias.Name, alias.Type.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Writer) write(s string) error {
	_, err := w.Writer.Write([]byte(s))
	return err
}

func (w *Writer) writeUnion(parent zng.Value) error {
	utyp := zng.AliasedType(parent.Type).(*zng.TypeUnion)
	inner, index, v, err := utyp.SplitZng(parent.Bytes)
	if err != nil {
		return err
	}
	s := strconv.FormatInt(index, 10) + ":"
	if err := w.write(s); err != nil {
		return err
	}

	value := zng.Value{inner, v}
	if zng.IsContainerType(zng.AliasedType(inner)) {
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
	realType := zng.AliasedType(parent.Type)
	if _, ok := realType.(*zng.TypeUnion); ok {
		return w.writeUnion(parent)
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
		for it := zcode.Iter(parent.Bytes); !it.Done(); {
			v, container, err := it.Next()
			if err != nil {
				return err
			}
			if columns != nil {
				if k >= len(columns) {
					return &zng.RecordTypeError{Name: "<record>", Type: parent.Type.String(), Err: zng.ErrExtraField}
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
	if err := w.write(v.Format(zng.OutFormatZNG)); err != nil {
		return err
	}
	return w.write(";")
}
