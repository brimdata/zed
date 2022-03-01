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

func formatAny(zv zed.Value, inContainer bool) string {
	switch t := zv.Type.(type) {
	case *zed.TypeArray:
		return formatArray(t, zv.Bytes)
	case *zed.TypeNamed:
		return formatAny(zed.Value{t.Type, zv.Bytes}, inContainer)
	case *zed.TypeOfBool:
		if zed.DecodeBool(zv.Bytes) {
			return "T"
		}
		return "F"
	case *zed.TypeOfBytes:
		return base64.StdEncoding.EncodeToString(zv.Bytes)
	case *zed.TypeOfDuration:
		// This format of a fractional second is used by Zeek in logs.
		// It uses enough precision to fully represent the 64-bit ns
		// accuracy of a nano Duration. Such values cannot be represented by
		// float64's without loss of the least significant digits of ns.
		return nano.Ts(zed.DecodeDuration(zv.Bytes)).StringFloat()
	case *zed.TypeEnum:
		return formatAny(zed.Value{zed.TypeUint64, zv.Bytes}, false)
	case *zed.TypeOfFloat32:
		return strconv.FormatFloat(float64(zed.DecodeFloat32(zv.Bytes)), 'f', -1, 32)
	case *zed.TypeOfFloat64:
		return strconv.FormatFloat(zed.DecodeFloat64(zv.Bytes), 'f', -1, 64)
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64:
		return strconv.FormatInt(zed.DecodeInt(zv.Bytes), 10)
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32, *zed.TypeOfUint64:
		return strconv.FormatUint(zed.DecodeUint(zv.Bytes), 10)
	case *zed.TypeOfIP:
		return zed.DecodeIP(zv.Bytes).String()
	case *zed.TypeMap:
		return formatMap(t, zv.Bytes)
	case *zed.TypeOfNet:
		return zed.DecodeNet(zv.Bytes).String()
	case *zed.TypeOfNull:
		return "-"
	case *zed.TypeRecord:
		return formatRecord(t, zv.Bytes)
	case *zed.TypeSet:
		return formatSet(t, zv.Bytes)
	case *zed.TypeOfString:
		return formatString(t, zv.Bytes, inContainer)
	case *zed.TypeOfTime:
		// This format of a fractional second is used by Zeek in logs.
		// It uses enough precision to fully represent the 64-bit ns
		// accuracy of a nano.Ts.  Such values cannot be representd by
		// float64's without loss of the least significant digits of ns.
		return zed.DecodeTime(zv.Bytes).StringFloat()
	case *zed.TypeOfType:
		return zson.String(zv)
	case *zed.TypeUnion:
		return formatUnion(t, zv.Bytes)
	case *zed.TypeError:
		if zed.TypeUnder(t.Type) == zed.TypeString {
			return string(zv.Bytes)
		}
		return zson.MustFormatValue(zv)
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
			b.WriteString(formatAny(zed.Value{t.Type, val}, true))
		}
	}
	return b.String()
}

func formatMap(t *zed.TypeMap, zv zcode.Bytes) string {
	var b strings.Builder
	it := zv.Iter()
	b.WriteByte('[')
	for !it.Done() {
		b.WriteString(formatAny(zed.Value{t.KeyType, it.Next()}, true))
		b.WriteString(formatAny(zed.Value{t.ValType, it.Next()}, true))
	}
	b.WriteByte(']')
	return b.String()
}

func formatRecord(t *zed.TypeRecord, zv zcode.Bytes) string {
	var b strings.Builder
	separator := byte(',')
	first := true
	it := zv.Iter()
	for _, col := range t.Columns {
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		if val := it.Next(); val == nil {
			b.WriteByte('-')
		} else {
			b.WriteString(formatAny(zed.Value{col.Type, val}, false))
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
		b.WriteString(formatAny(zed.Value{t.Type, it.Next()}, true))
	}
	return b.String()
}

func formatString(t *zed.TypeOfString, zv zcode.Bytes, inContainer bool) string {
	if bytes.Equal(zv, []byte{'-'}) {
		return "\\x2d"
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
		return FormatValue(zed.Value{zed.TypeNull, nil})
	}
	typ, iv := t.SplitZNG(zv)
	s := strconv.FormatInt(int64(t.Selector(typ)), 10) + ":"
	return s + formatAny(zed.Value{typ, iv}, false)
}

func FormatValue(v zed.Value) string {
	if v.Bytes == nil {
		return "-"
	}
	return formatAny(v, false)
}
