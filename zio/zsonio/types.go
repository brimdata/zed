package zsonio

import (
	"github.com/brimsec/zq/zng"
)

func (w *Writer) writeType(indent int, typ zng.Type) error {
	if w.typeTab == 0 {
		indent = 0
	}
	switch t := typ.(type) {
	//XXX Need enum support. See #1676.
	default:
		return w.write(typ.String())
	case *zng.TypeAlias:
		// If the type alias here isn't yet defined in the local
		// output context, we need to define it by looking it up the
		// in the foreign context and decorating the type definition
		// here with the alias name if needed. XXX For now, we just
		// print the alias name here even if it's not defined in the
		// local context (we can't look it up in the local context
		// since it may not exist).  See issue #1675.
		return w.write(t.Name)
	case *zng.TypeRecord:
		return w.writeTypeRecord(indent, t)
	case *zng.TypeArray:
		return w.writeTypeVector(indent, "[", "]", t.Type)
	case *zng.TypeSet:
		return w.writeTypeVector(indent, "|[", "]|", t.Type)
	case *zng.TypeUnion:
		return w.writeTypeUnion(indent, t)
	case *zng.TypeMap:
		return w.writeTypeMap(indent, t)
	}
}

func (w *Writer) writeTypeRecord(indent int, typ *zng.TypeRecord) error {
	if err := w.write("{"); err != nil {
		return err
	}
	if len(typ.Columns) == 0 {
		return w.write("}")
	}
	indent += w.typeTab
	sep := w.typeNewline
	for _, field := range typ.Columns {
		if err := w.write(sep); err != nil {
			return err
		}
		if err := w.indent(indent, field.Name); err != nil {
			return err
		}
		if err := w.write(": "); err != nil {
			return err
		}
		if err := w.writeType(indent, field.Type); err != nil {
			return err
		}
		sep = "," + w.typeNewline
	}
	if err := w.write(w.typeNewline); err != nil {
		return err
	}
	return w.indent(indent-w.typeTab, "}")
}

func (w *Writer) writeTypeVector(indent int, open, close string, inner zng.Type) error {
	if err := w.write(open); err != nil {
		return err
	}
	indent += w.typeTab
	sep := w.typeNewline
	if err := w.write(sep); err != nil {
		return err
	}
	if err := w.indent(indent, ""); err != nil {
		return err
	}
	if err := w.writeType(indent, inner); err != nil {
		return err
	}
	if err := w.write(w.typeNewline); err != nil {
		return err
	}
	return w.indent(indent-w.typeTab, close)
}

func (w *Writer) writeTypeUnion(indent int, union *zng.TypeUnion) error {
	if err := w.write("["); err != nil {
		return err
	}
	indent += w.typeTab
	sep := w.typeNewline
	for _, typ := range union.Types {
		if err := w.write(sep); err != nil {
			return err
		}
		if err := w.indent(indent, ""); err != nil {
			return err
		}
		if err := w.writeType(indent, typ); err != nil {
			return err
		}
		if err := w.write(w.typeNewline); err != nil {
			return err
		}
		sep = "," + w.typeNewline
	}
	if err := w.write(w.typeNewline); err != nil {
		return err
	}
	return w.indent(indent-w.typeTab, "]")
}

func (w *Writer) writeTypeMap(indent int, typ *zng.TypeMap) error {
	if err := w.write("|{"); err != nil {
		return err
	}
	indent += w.typeTab
	sep := w.typeNewline
	for _, typ := range []zng.Type{typ.KeyType, typ.ValType} {
		if err := w.write(sep); err != nil {
			return err
		}
		if err := w.writeType(indent+w.typeTab, typ); err != nil {
			return err
		}
		sep = "," + w.typeNewline
	}
	if err := w.write(w.typeNewline); err != nil {
		return err
	}
	return w.indent(indent-w.typeTab, "}|")
}
