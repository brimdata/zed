package zeekio

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

func formatAny(val zed.Value, inContainer bool) string {
	switch t := val.Type().(type) {
	case *zed.TypeArray:
		return formatArray(t, val.Bytes())
	case *zed.TypeNamed:
		return formatAny(zed.NewValue(t.Type, val.Bytes()), inContainer)
	case *zed.TypeOfBool:
		if val.Bool() {
			return "T"
		}
		return "F"
	case *zed.TypeOfBytes:
		return base64.StdEncoding.EncodeToString(val.Bytes())
	case *zed.TypeOfDuration, *zed.TypeOfTime:
		return formatTime(nano.Ts(val.Int()))
	case *zed.TypeEnum:
		return formatAny(zed.NewValue(zed.TypeUint64, val.Bytes()), false)
	case *zed.TypeOfFloat16, *zed.TypeOfFloat32:
		return strconv.FormatFloat(val.Float(), 'f', -1, 32)
	case *zed.TypeOfFloat64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64)
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64:
		return strconv.FormatInt(val.Int(), 10)
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32, *zed.TypeOfUint64:
		return strconv.FormatUint(val.Uint(), 10)
	case *zed.TypeOfIP:
		return zed.DecodeIP(val.Bytes()).String()
	case *zed.TypeMap:
		return formatMap(t, val.Bytes())
	case *zed.TypeOfNet:
		return zed.DecodeNet(val.Bytes()).String()
	case *zed.TypeOfNull:
		return "-"
	case *zed.TypeRecord:
		return formatRecord(t, val.Bytes())
	case *zed.TypeSet:
		return formatSet(t, val.Bytes())
	case *zed.TypeOfString:
		return formatString(t, val.Bytes(), inContainer)
	case *zed.TypeOfType:
		return zson.String(val)
	case *zed.TypeUnion:
		return formatUnion(t, val.Bytes())
	case *zed.TypeError:
		if zed.TypeUnder(t.Type) == zed.TypeString {
			return string(val.Bytes())
		}
		return zson.FormatValue(val)
	default:
		return fmt.Sprintf("zeekio.StringOf(): unknown type: %T", t)
	}
}

func formatArray(t *zed.TypeArray, zv zcode.Bytes) string {
	if len(zv) == 0 {
		return "(empty)"
	}

	var b strings.Builder
	separator := byte(',')

	first := true
	it := zv.Iter()
	for !it.Done() {
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		if val := it.Next(); val == nil {
			b.WriteByte('-')
		} else {
			b.WriteString(formatAny(zed.NewValue(t.Type, val), true))
		}
	}
	return b.String()
}

func formatMap(t *zed.TypeMap, zv zcode.Bytes) string {
	var b strings.Builder
	it := zv.Iter()
	b.WriteByte('[')
	for !it.Done() {
		b.WriteString(formatAny(zed.NewValue(t.KeyType, it.Next()), true))
		b.WriteString(formatAny(zed.NewValue(t.ValType, it.Next()), true))
	}
	b.WriteByte(']')
	return b.String()
}

func formatRecord(t *zed.TypeRecord, zv zcode.Bytes) string {
	var b strings.Builder
	separator := byte(',')
	first := true
	it := zv.Iter()
	for _, f := range t.Fields {
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		if val := it.Next(); val == nil {
			b.WriteByte('-')
		} else {
			b.WriteString(formatAny(zed.NewValue(f.Type, val), false))
		}
	}
	return b.String()
}

func formatSet(t *zed.TypeSet, zv zcode.Bytes) string {
	if len(zv) == 0 {
		return "(empty)"
	}
	var b strings.Builder
	separator := byte(',')
	first := true
	it := zv.Iter()
	for !it.Done() {
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		b.WriteString(formatAny(zed.NewValue(t.Type, it.Next()), true))
	}
	return b.String()
}

func formatString(t *zed.TypeOfString, zv zcode.Bytes, inContainer bool) string {
	if bytes.Equal(zv, []byte{'-'}) {
		return "\\x2d"
	}
	if string(zv) == "(empty)" {
		return "\\x28empty)"
	}
	var out []byte
	var start int
	for i := 0; i < len(zv); {
		r, l := utf8.DecodeRune(zv[i:])
		if r == '\\' {
			out = append(out, zv[start:i]...)
			out = append(out, '\\', '\\')
			i++
			start = i
			continue
		}
		if !unicode.IsPrint(r) || shouldEscape(r, inContainer) {
			out = append(out, zv[start:i]...)
			out = append(out, unescape(r)...)
			i += l
			start = i
		} else {
			i += l
		}
	}
	return string(append(out, zv[start:]...))
}

func unescape(r rune) []byte {
	code := strconv.FormatInt(int64(r), 16)
	n := len(code)
	if (n & 1) != 0 {
		n++
		code = "0" + code
	}
	var b bytes.Buffer
	for k := 0; k < n; k += 2 {
		b.WriteString("\\x")
		b.WriteString(code[k : k+2])
	}
	return b.Bytes()
}

func formatUnion(t *zed.TypeUnion, zv zcode.Bytes) string {
	if zv == nil {
		return FormatValue(zed.Null)
	}
	typ, iv := t.Untag(zv)
	s := strconv.FormatInt(int64(t.TagOf(typ)), 10) + ":"
	return s + formatAny(zed.NewValue(typ, iv), false)
}

func FormatValue(v zed.Value) string {
	if v.IsNull() {
		return "-"
	}
	return formatAny(v, false)
}
