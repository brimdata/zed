package zeekio

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/flattener"
	"github.com/brimsec/zq/zng/resolver"
)

var ErrDescriptorChanged = errors.New("descriptor changed")

type Writer struct {
	writer io.WriteCloser
	header
	flattener *flattener.Flattener
	typ       *zng.TypeRecord
	precision int
	utf8      bool
}

func NewWriter(w io.WriteCloser, utf8 bool) *Writer {
	return &Writer{
		writer:    w,
		flattener: flattener.New(resolver.NewContext()),
		precision: 6,
		utf8:      utf8,
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
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
	values, changePrecision, err := ZeekStrings(r, w.precision, w.utf8)
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
	_, err = w.writer.Write([]byte(out))
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
	_, err := w.writer.Write([]byte(s))
	return err
}

func isHighPrecision(ts nano.Ts) bool {
	_, ns := ts.Split()
	return (ns/1000)*1000 != ns
}

// This returns the zeek strings for this record.  XXX We need to not use this.
// XXX change to Pretty for output writers?... except zeek?
func ZeekStrings(r *zng.Record, precision int, utf8 bool) ([]string, bool, error) {
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
			var err error
			field, err = format(col.Type, val)
			if err != nil {
				return nil, false, err
			}
			if !utf8 {
				field = EscapeNonASCII(field)
			}
		}
		ss = append(ss, field)
	}
	return ss, changePrecision, nil
}

func format(typ zng.Type, zb zcode.Bytes) (string, error) {
	if zb == nil {
		return "-", nil
	}
	switch typ := zng.AliasedType(typ).(type) {
	case *zng.TypeRecord, *zng.TypeUnion, *zng.TypeEnum, *zng.TypeMap:
		return "", fmt.Errorf("can't format type %T", typ)
	case *zng.TypeArray:
		return formatArrayOrSet(typ.Type, zb)
	case *zng.TypeSet:
		return formatArrayOrSet(typ.Type, zb)
	default:
		return typ.StringOf(zb, zng.OutFormatUnescaped, false), nil
	}
}

func formatArrayOrSet(typ zng.Type, zb zcode.Bytes) (string, error) {
	iter := zb.Iter()
	if iter.Done() {
		return "(empty)", nil
	}
	var b strings.Builder
	for {
		zb, _, err := iter.Next()
		if err != nil {
			return "", err
		}
		s, err := format(typ, zb)
		if err != nil {
			return "", err
		}
		b.WriteString(strings.ReplaceAll(s, ",", "\\x2c"))
		if iter.Done() {
			return b.String(), nil
		}
		b.WriteByte(',')
	}
}

// EscapeNonASCII returns a copy of s with each non-ASCII byte replaced by its
// corresponding \xhh hexadecimal escape sequence.
func EscapeNonASCII(s string) string {
	var nonascii bool
	for i := 0; i < len(s); i++ {
		if rune(s[i]) > unicode.MaxASCII {
			nonascii = true
			break
		}
	}
	if !nonascii {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if rune(s[i]) > unicode.MaxASCII {
			b.WriteString("\\x")
			const hex = "0123456789abcdef"
			b.WriteByte(hex[s[i]>>4])
			b.WriteByte(hex[s[i]&0xf])
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
