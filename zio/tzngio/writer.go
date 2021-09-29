package tzngio

import (
	"fmt"
	"io"
	"strconv"

	"github.com/brimdata/zed"
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

func (w *Writer) Write(r *zed.Record) error {
	inID := r.Type.ID()
	name, ok := w.tracker[inID]
	if !ok {
		if err := w.writeAliases(r.Type); err != nil {
			return err
		}
		typ := r.Type
		var op string
		if alias, ok := typ.(*zed.TypeAlias); ok {
			name = alias.Name
			op = "="
		} else {
			id := len(w.tracker)
			name = strconv.Itoa(id)
			op = ":"
		}
		w.tracker[inID] = name
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

func (w *Writer) writeAliases(typ zed.Type) error {
	aliases := findAliases(typ)
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

func findAliases(typ zed.Type) []*zed.TypeAlias {
	var aliases []*zed.TypeAlias
	switch typ := typ.(type) {
	case *zed.TypeSet:
		aliases = findAliases(typ.Type)
	case *zed.TypeArray:
		aliases = findAliases(typ.Type)
	case *zed.TypeRecord:
		for _, col := range typ.Columns {
			aliases = append(aliases, findAliases(col.Type)...)
		}
	case *zed.TypeUnion:
		for _, typ := range typ.Types {
			aliases = append(aliases, findAliases(typ)...)
		}
	case *zed.TypeMap:
		keyAliases := findAliases(typ.KeyType)
		valAliases := findAliases(typ.KeyType)
		aliases = append(keyAliases, valAliases...)
	case *zed.TypeAlias:
		aliases = append(aliases, findAliases(typ.Type)...)
		aliases = append(aliases, typ)
	}
	return aliases
}

func (w *Writer) write(s string) error {
	_, err := w.writer.Write([]byte(s))
	return err
}

func (w *Writer) writeUnion(parent zed.Value) error {
	utyp := zed.AliasOf(parent.Type).(*zed.TypeUnion)
	inner, selector, v, err := utyp.SplitZng(parent.Bytes)
	if err != nil {
		return err
	}
	s := strconv.FormatInt(selector, 10) + ":"
	if err := w.write(s); err != nil {
		return err
	}

	value := zed.Value{inner, v}
	if zed.IsContainerType(zed.AliasOf(inner)) {
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

func (w *Writer) writeContainer(parent zed.Value) error {
	if parent.IsUnsetOrNil() {
		w.write("-;")
		return nil
	}
	realType := zed.AliasOf(parent.Type)
	if _, ok := realType.(*zed.TypeUnion); ok {
		return w.writeUnion(parent)
	}
	if typ, ok := realType.(*zed.TypeMap); ok {
		s := StringOf(zed.Value{typ, parent.Bytes}, OutFormatZNG, true)
		return w.write(s)
	}
	if err := w.write("["); err != nil {
		return err
	}
	childType, columns := zed.ContainedType(realType)
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
					return &zed.RecordTypeError{Name: "<record>", Type: parent.Type.String(), Err: zed.ErrExtraField}
				}
				childType = columns[k].Type
				k++
			}
			value := zed.Value{childType, v}
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

func (w *Writer) writeValue(v zed.Value) error {
	if v.IsUnsetOrNil() {
		return w.write("-;")
	}
	if err := w.write(FormatValue(v, OutFormatZNG)); err != nil {
		return err
	}
	return w.write(";")
}
