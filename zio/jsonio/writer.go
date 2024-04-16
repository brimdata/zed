package jsonio

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/terminal/color"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

var (
	boolColor   = []byte("\x1b[1m")
	fieldColor  = []byte("\x1b[34;1m")
	nullColor   = []byte("\x1b[2m")
	numberColor = []byte("\x1b[36m")
	puncColor   = []byte{} // no color
	stringColor = []byte("\x1b[32m")
)

type Writer struct {
	io.Closer
	writer *bufio.Writer
	tab    int

	// Use json.Encoder for primitive Values. Have to use
	// json.Encoder instead of json.Marshal because it's
	// the only way to turn off HTML escaping.
	primEnc *json.Encoder
	primBuf bytes.Buffer
}

type WriterOpts struct {
	Pretty int
}

func NewWriter(writer io.WriteCloser, opts WriterOpts) *Writer {
	w := &Writer{
		Closer: writer,
		writer: bufio.NewWriter(writer),
		tab:    opts.Pretty,
	}
	w.primEnc = json.NewEncoder(&w.primBuf)
	w.primEnc.SetEscapeHTML(false)
	return w
}

func (w *Writer) Write(val zed.Value) error {
	// writeAny doesn't return an error because any error that occurs will be
	// surfaced with w.writer.Flush is called.
	w.writeAny(0, val)
	w.writer.WriteByte('\n')
	return w.writer.Flush()
}

func (w *Writer) writeAny(tab int, val zed.Value) {
	val = val.Under()
	if val.IsNull() {
		w.writeColor([]byte("null"), nullColor)
		return
	}
	if val.Type().ID() < zed.IDTypeComplex {
		w.writePrimitive(val)
		return
	}
	switch typ := val.Type().(type) {
	case *zed.TypeRecord:
		w.writeRecord(tab, typ, val.Bytes())
	case *zed.TypeArray:
		w.writeArray(tab, typ.Type, val.Bytes())
	case *zed.TypeSet:
		w.writeArray(tab, typ.Type, val.Bytes())
	case *zed.TypeMap:
		w.writeMap(tab, typ, val.Bytes())
	case *zed.TypeEnum:
		w.writeEnum(typ, val.Bytes())
	case *zed.TypeError:
		w.writeError(tab, typ, val.Bytes())
	default:
		panic(fmt.Sprintf("unsupported type: %s", zson.FormatType(typ)))
	}
}

func (w *Writer) writeRecord(tab int, typ *zed.TypeRecord, bytes zcode.Bytes) {
	tab += w.tab
	w.punc('{')
	it := bytes.Iter()
	for i, f := range typ.Fields {
		if i != 0 {
			w.punc(',')
		}
		w.writeEntry(tab, f.Name, zed.NewValue(f.Type, it.Next()))
	}
	w.newline()
	w.indent(tab - w.tab)
	w.punc('}')
}

func (w *Writer) writeArray(tab int, typ zed.Type, bytes zcode.Bytes) {
	tab += w.tab
	w.punc('[')
	it := bytes.Iter()
	for i := 0; !it.Done(); i++ {
		if i != 0 {
			w.punc(',')
		}
		w.newline()
		w.indent(tab)
		w.writeAny(tab, zed.NewValue(typ, it.Next()))
	}
	w.newline()
	w.indent(tab - w.tab)
	w.punc(']')
}

func (w *Writer) writeMap(tab int, typ *zed.TypeMap, bytes zcode.Bytes) {
	tab += w.tab
	w.punc('{')
	it := bytes.Iter()
	for i := 0; !it.Done(); i++ {
		if i != 0 {
			w.punc(',')
		}
		key := mapKey(typ.KeyType, it.Next())
		w.writeEntry(tab, key, zed.NewValue(typ.ValType, it.Next()))
	}
	w.newline()
	w.indent(tab - w.tab)
	w.punc('}')
}

func mapKey(typ zed.Type, b zcode.Bytes) string {
	val := zed.NewValue(typ, b)
	switch val.Type().Kind() {
	case zed.PrimitiveKind:
		if val.Type().ID() == zed.IDString {
			// Don't quote strings.
			return val.AsString()
		}
		return zson.FormatPrimitive(val.Type(), val.Bytes())
	case zed.UnionKind:
		// Untagged, decorated ZSON so
		// |{0:1,0(uint64):2,0(=t):3,"0":4}| gets unique keys.
		typ, bytes := typ.(*zed.TypeUnion).Untag(b)
		return zson.FormatValue(zed.NewValue(typ, bytes))
	case zed.EnumKind:
		return convertEnum(typ.(*zed.TypeEnum), b)
	default:
		return zson.FormatValue(val)
	}
}

func (w *Writer) writeEnum(typ *zed.TypeEnum, bytes zcode.Bytes) {
	w.writeColor(w.marshalJSON(convertEnum(typ, bytes)), stringColor)
}

func convertEnum(typ *zed.TypeEnum, bytes zcode.Bytes) string {
	if k := int(zed.DecodeUint(bytes)); k < len(typ.Symbols) {
		return typ.Symbols[k]
	}
	return "<bad enum>"
}

func (w *Writer) writeError(tab int, typ *zed.TypeError, bytes zcode.Bytes) {
	tab += w.tab
	w.punc('{')
	w.writeEntry(tab, "error", zed.NewValue(typ.Type, bytes))
	w.newline()
	w.indent(tab - w.tab)
	w.punc('}')
}

func (w *Writer) writeEntry(tab int, name string, val zed.Value) {
	w.newline()
	w.indent(tab)
	w.writeColor(w.marshalJSON(name), fieldColor)
	w.punc(':')
	if w.tab != 0 {
		w.writer.WriteByte(' ')
	}
	w.writeAny(tab, val)
}

func (w *Writer) writePrimitive(val zed.Value) {
	var v any
	c := stringColor
	switch id := val.Type().ID(); {
	case id == zed.IDDuration:
		v = nano.Duration(val.Int()).String()
	case id == zed.IDTime:
		v = nano.Ts(val.Int()).Time().Format(time.RFC3339Nano)
	case zed.IsSigned(id):
		v, c = val.Int(), numberColor
	case zed.IsUnsigned(id):
		v, c = val.Uint(), numberColor
	case zed.IsFloat(id):
		v, c = val.Float(), numberColor
	case id == zed.IDBool:
		v, c = val.AsBool(), boolColor
	case id == zed.IDBytes:
		v = "0x" + hex.EncodeToString(val.Bytes())
	case id == zed.IDString:
		v = val.AsString()
	case id == zed.IDIP:
		v = zed.DecodeIP(val.Bytes()).String()
	case id == zed.IDNet:
		v = zed.DecodeNet(val.Bytes()).String()
	case id == zed.IDType:
		v = zson.FormatValue(val)
	default:
		panic(fmt.Sprintf("unsupported id=%d", id))
	}
	w.writeColor(w.marshalJSON(v), c)
}

func (w *Writer) marshalJSON(v any) []byte {
	w.primBuf.Reset()
	if err := w.primEnc.Encode(v); err != nil {
		panic(err)
	}
	return bytes.TrimSpace(w.primBuf.Bytes())
}

func (w *Writer) punc(b byte) {
	w.writeColor([]byte{b}, puncColor)
}

func (w *Writer) writeColor(b []byte, code []byte) {
	if w.tab > 0 && color.Enabled {
		w.writer.Write(code)
		defer w.writer.WriteString(color.Reset.String())
	}
	w.writer.Write(b)
}

func (w *Writer) newline() {
	if w.tab > 0 {
		w.writer.WriteByte('\n')
	}
}

func (w *Writer) indent(tab int) {
	w.writer.Write(bytes.Repeat([]byte(" "), tab))
}
