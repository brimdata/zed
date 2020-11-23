package zsonio

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Writer struct {
	writer      io.WriteCloser
	zctx        *resolver.Context
	mapper      *resolver.Mapper
	tags        typemap
	tab         int
	newline     string
	whitespace  string
	typeTab     int
	typeNewline string
}

type WriterOpts struct {
	Pretty int
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	newline := ""
	if opts.Pretty > 0 {
		newline = "\n"
	}
	zctx := resolver.NewContext()
	return &Writer{
		zctx:       zctx,
		writer:     w,
		tags:       make(typemap),
		tab:        opts.Pretty,
		newline:    newline,
		whitespace: strings.Repeat(" ", 80),
		mapper:     resolver.NewMapper(zctx),
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(rec *zng.Record) error {
	typ, err := w.mapper.Translate(rec.Type)
	if err != nil {
		return err
	}
	if err := w.writeValue(0, typ, typ, rec.Raw); err != nil {
		return err
	}
	return w.write(",\n")
}

func (w *Writer) writeValue(indent int, typ, decType zng.Type, bytes zcode.Bytes) error {
	if bytes == nil {
		if err := w.write("null"); err != nil {
			if err != nil {
				return err
			}
		}
		return w.writeDecorator(decType)
	}
	var err error
	switch t := typ.(type) {
	//XXX Need enum support. See #1676.
	default:
		err = w.writePrimitive(indent, typ, bytes)
	case *zng.TypeAlias:
		if decType == nil {
			return w.writeValue(indent, t.Type, nil, bytes)
		}
		if w.tags.exists(typ) {
			return w.writeValue(indent, t.Type, typ, bytes)
		}
		if err := w.writeValue(indent, t.Type, t.Type, bytes); err != nil {
			return err
		}
		w.tags.enter(typ, t.Name)
		decType = typ
	case *zng.TypeRecord:
		err = w.writeRecord(indent, t, bytes)
	case *zng.TypeArray:
		err = w.writeVector(indent, "[", "]", t.Type, zng.Value{t, bytes})
		if decType == nil {
			return err
		}
		decType = t.Type
	case *zng.TypeSet:
		err = w.writeVector(indent, "|[", "]|", t.Type, zng.Value{t, bytes})
		if decType == nil {
			return err
		}
		decType = t.Type
	case *zng.TypeUnion:
		if err := w.writeUnion(indent, t, decType, bytes); err != nil {
			return err
		}
		if w.tags.exists(typ) {
			return nil
		}
		w.tags.enter(typ, strconv.Itoa(t.ID()))
	case *zng.TypeMap:
		if err := w.writeMap(indent, t, bytes); err != nil {
			return err
		}
		if !w.tags.exists(typ) {
			if err := w.writeDecorator(decType); err != nil {
				return err
			}
			w.tags.enter(typ, strconv.Itoa(t.ID()))
		}
	}
	if err != nil {
		return err
	}
	return w.writeDecorator(decType)
}

func (w *Writer) writeDecorator(typ zng.Type) error {
	if typ == nil || impliedPrimitive(typ) {
		return nil
	}
	if !w.tags.exists(typ) {
		w.tags.enter(typ, w.tags.lookup(typ))
	}
	return w.writef(" (%s)", w.tags.lookup(typ))
}

func (w *Writer) writeRecord(indent int, typ *zng.TypeRecord, bytes zcode.Bytes) error {
	if err := w.write("{"); err != nil {
		return err
	}
	if len(typ.Columns) == 0 {
		return w.write("}")
	}
	seen := w.tags.exists(typ)
	indent += w.tab
	sep := w.newline
	it := bytes.Iter()
	for _, field := range typ.Columns {
		if it.Done() {
			return &zng.RecordTypeError{Name: string(field.Name), Type: field.Type.String(), Err: zng.ErrMissingField}
		}
		bytes, _, err := it.Next()
		if err != nil {
			return err
		}
		if err := w.write(sep); err != nil {
			return err
		}
		if err := w.indent(indent, field.Name); err != nil {
			return err
		}
		if err := w.write(": "); err != nil {
			return err
		}
		var decType zng.Type
		if !seen {
			decType = field.Type
		}
		if err := w.writeValue(indent, field.Type, decType, bytes); err != nil {
			return err
		}
		sep = "," + w.newline
	}
	if err := w.write(w.newline); err != nil {
		return err
	}
	return w.indent(indent-w.tab, "}")
}

func (w *Writer) writeVector(indent int, open, close string, inner zng.Type, zv zng.Value) error {
	if err := w.write(open); err != nil {
		return err
	}
	len, err := zv.ContainerLength()
	if err != nil {
		return err
	}
	if len == 0 {
		return w.write(close)
	}
	indent += w.tab
	sep := w.newline
	it := zv.Iter()
	for !it.Done() {
		bytes, _, err := it.Next()
		if err != nil {
			return err
		}
		if err := w.write(sep); err != nil {
			return err
		}
		if err := w.indent(indent, ""); err != nil {
			return err
		}
		if err := w.writeValue(indent, inner, nil, bytes); err != nil {
			return err
		}
		sep = "," + w.newline
	}
	if err := w.write(w.newline); err != nil {
		return err
	}
	return w.indent(indent-w.tab, close)
}

func (w *Writer) writeUnion(indent int, union *zng.TypeUnion, decType zng.Type, bytes zcode.Bytes) error {
	typ, selector, bytes, err := union.SplitZng(bytes)
	if err != nil {
		return err
	}
	if err := w.writef("<%d>", selector); err != nil {
		return err
	}
	return w.writeValue(indent, typ, decType, bytes)
}

func (w *Writer) writeMap(indent int, typ *zng.TypeMap, bytes zcode.Bytes) error {
	if err := w.write("|{"); err != nil {
		return err
	}
	if bytes == nil {
		return w.write("|}")
	}
	indent += w.tab
	sep := w.newline
	for it := bytes.Iter(); !it.Done(); {
		keyBytes, _, err := it.Next()
		if err != nil {
			return err
		}
		if it.Done() {
			return errors.New("truncated map value")
		}
		valBytes, _, err := it.Next()
		if err != nil {
			return err
		}
		if err := w.write(sep); err != nil {
			return err
		}
		if err := w.indent(indent, "{"); err != nil {
			return err
		}
		if err := w.writeValue(indent+w.tab, typ.KeyType, nil, keyBytes); err != nil {
			return err
		}
		if err := w.write(","); err != nil {
			return err
		}
		if err := w.writeValue(indent+w.tab, typ.ValType, nil, valBytes); err != nil {
			return err
		}
		if err := w.write("}"); err != nil {
			return err
		}
		sep = "," + w.newline
	}
	if err := w.write(w.newline); err != nil {
		return err
	}
	return w.indent(indent-w.tab, "}|")
}

func (w *Writer) writePrimitive(indent int, typ zng.Type, bytes zcode.Bytes) error {
	switch typ.(type) {
	default:
		zv := zng.Value{typ, bytes} //XXX
		if err := w.write(zv.Format(zng.OutFormatZNG)); err != nil {
			return err
		}
	case *zng.TypeOfTime:
		t, err := zng.DecodeTime(bytes)
		if err != nil {
			return err
		}
		b := t.Time().Format(time.RFC3339Nano)
		return w.write(string(b))

	case *zng.TypeOfBytes:
		if err := w.write("0x"); err != nil {
			return err
		}
		if err := w.write(string(hex.EncodeToString(bytes))); err != nil {
			return err
		}
	case *zng.TypeOfString, *zng.TypeOfBstring, *zng.TypeOfError:
		if err := w.write("\""); err != nil {
			return err
		}
		//XXX Need to properly escape quoted string (issue #1677)
		zv := zng.Value{typ, bytes} //XXX
		if err := w.write(zv.Format(zng.OutFormatZNG)); err != nil {
			return err
		}
		if err := w.write("\""); err != nil {
			return err
		}
	case *zng.TypeOfType:
		// XXX This should change to a lookup in the foreign context
		// using the zcode.Bytes as the key, and when there's a miss,
		// allocating the type in the local context, all while handling
		// aliases.  See issue #1675.
		typ, err := w.zctx.LookupByName(string(bytes))
		if err != nil {
			//return err
			// XXX Until #1675 is resolved, emit the old
			// TZNG type name and flag it as problematic.
			w.write("TZNG[")
			w.write(string(bytes))
			w.write("]")
			return nil

		}
		if w.typeTab == 0 {
			indent = 0
		}
		return w.writeType(indent, typ)

	}
	return nil
}

func (w *Writer) indent(tab int, s string) error {
	n := len(w.whitespace)
	if n < tab {
		n = 2 * tab
		w.whitespace = strings.Repeat(" ", n)
	}
	if err := w.write(w.whitespace[0:tab]); err != nil {
		return err
	}
	return w.write(s)
}

func (w *Writer) write(s string) error {
	_, err := w.writer.Write([]byte(s))
	return err
}

func (w *Writer) writef(s string, args ...interface{}) error {
	_, err := fmt.Fprintf(w.writer, s, args...)
	return err
}

func impliedPrimitive(typ zng.Type) bool {
	switch typ.(type) {
	case *zng.TypeOfInt64, *zng.TypeOfTime, *zng.TypeOfFloat64, *zng.TypeOfBool, *zng.TypeOfBytes, *zng.TypeOfString, *zng.TypeOfIP, *zng.TypeOfNet:
		return true
	}
	return false
}
