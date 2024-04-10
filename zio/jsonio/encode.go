package jsonio

import (
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

type encoder struct {
	w           io.Writer
	buf         bytes.Buffer
	tab         int
	newlineChar string

	// Use json.Encoder to get marshal primitive Values. Have to use
	// json.Encoder instead of json.Marshal because this is the only way to get
	// pkg json to turn off HTML escaping.
	primEnc *json.Encoder
	primBuf bytes.Buffer
}

func newEncoder(w io.Writer, indent int) *encoder {
	e := &encoder{w: w, tab: indent}
	e.primEnc = json.NewEncoder(&e.primBuf)
	e.primEnc.SetEscapeHTML(false)
	if indent > 0 {
		e.newlineChar = "\n"
	}
	return e
}

func (e *encoder) encodeVal(val zed.Value) error {
	e.buf.Reset()
	e.marshal(0, val)
	e.buf.WriteByte('\n')
	_, err := e.w.Write(e.buf.Bytes())
	return err
}

func (e *encoder) marshal(tab int, val zed.Value) {
	val = val.Under()
	if val.IsNull() {
		e.writeColor([]byte("null"), nullColor)
		return
	}
	if val.Type().ID() < zed.IDTypeComplex {
		e.marshalPrimitive(val)
		return
	}
	switch typ := val.Type().(type) {
	case *zed.TypeRecord:
		e.marshalRecord(tab, typ, val.Bytes())
	case *zed.TypeArray:
		e.marshalArray(tab, typ.Type, val.Bytes())
	case *zed.TypeSet:
		e.marshalArray(tab, typ.Type, val.Bytes())
	case *zed.TypeMap:
		e.marshalMap(tab, typ, val.Bytes())
	case *zed.TypeEnum:
		e.marshalEnum(typ, val.Bytes())
	case *zed.TypeError:
		e.marshalError(tab, typ, val.Bytes())
	default:
		m := fmt.Sprintf("<unsupported type: %s>", zson.FormatType(typ))
		e.writeColor(e.marshalJSON(m), stringColor)
	}
}

func (e *encoder) marshalRecord(tab int, typ *zed.TypeRecord, bytes zcode.Bytes) {
	tab += e.tab
	e.punc('{')
	it := bytes.Iter()
	for i, f := range typ.Fields {
		if i != 0 {
			e.punc(',')
		}
		e.newline()
		e.indent(tab)
		e.writeColor(e.marshalJSON(f.Name), fieldColor)
		e.punc(':')
		if e.tab != 0 {
			e.buf.WriteByte(' ')
		}
		e.marshal(tab, zed.NewValue(f.Type, it.Next()))
	}
	e.newline()
	e.indent(tab - e.tab)
	e.punc('}')
}

func (e *encoder) marshalArray(tab int, typ zed.Type, bytes zcode.Bytes) {
	tab += e.tab
	e.punc('[')
	it := bytes.Iter()
	for i := 0; !it.Done(); i++ {
		if i != 0 {
			e.punc(',')
		}
		e.newline()
		e.indent(tab)
		e.marshal(tab, zed.NewValue(typ, it.Next()))
	}
	e.newline()
	e.indent(tab - e.tab)
	e.punc(']')
}

func (e *encoder) marshalMap(tab int, typ *zed.TypeMap, bytes zcode.Bytes) {
	tab += e.tab
	e.punc('{')
	it := bytes.Iter()
	for i := 0; !it.Done(); i++ {
		if i != 0 {
			e.punc(',')
		}
		e.newline()
		e.indent(tab)
		key := mapKey(typ.KeyType, it.Next())
		e.writeColor(e.marshalJSON(key), fieldColor)
		e.punc(':')
		if e.tab != 0 {
			e.buf.WriteByte(' ')
		}
		e.marshal(tab, zed.NewValue(typ.ValType, it.Next()))
	}
	e.newline()
	e.indent(tab - e.tab)
	e.punc('}')
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

func (e *encoder) marshalEnum(typ *zed.TypeEnum, bytes zcode.Bytes) {
	e.writeColor(e.marshalJSON(convertEnum(typ, bytes)), stringColor)
}

func convertEnum(typ *zed.TypeEnum, bytes zcode.Bytes) string {
	if k := int(zed.DecodeUint(bytes)); k < len(typ.Symbols) {
		return typ.Symbols[k]
	}
	return "<bad enum>"
}

func (e *encoder) marshalError(tab int, typ *zed.TypeError, bytes zcode.Bytes) {
	tab += e.tab
	e.punc('{')
	e.newline()
	e.indent(tab)
	e.writeColor([]byte(`"error"`), fieldColor)
	e.punc(':')
	if e.tab != 0 {
		e.buf.WriteByte(' ')
	}
	e.marshal(tab, zed.NewValue(typ.Type, bytes))
	e.newline()
	e.indent(tab - e.tab)
	e.punc('}')
}

func (e *encoder) marshalPrimitive(val zed.Value) {
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
		v = fmt.Sprintf("<unsupported id=%d>", id)
	}
	e.writeColor(e.marshalJSON(v), c)
}

func (e *encoder) marshalJSON(v any) []byte {
	e.primBuf.Reset()
	if err := e.primEnc.Encode(v); err != nil {
		panic(err)
	}
	return bytes.TrimSpace(e.primBuf.Bytes())
}

func (e *encoder) punc(b byte) {
	e.writeColor([]byte{b}, puncColor)
}

func (e *encoder) writeColor(b []byte, code []byte) {
	if e.tab > 0 && color.Enabled {
		e.buf.Write(code)
		defer e.buf.WriteString(color.Reset.String())
	}
	e.buf.Write(b)
}

func (e *encoder) newline() {
	e.buf.WriteString(e.newlineChar)
}

func (e *encoder) indent(tab int) {
	e.buf.Write(bytes.Repeat([]byte(" "), tab))
}
