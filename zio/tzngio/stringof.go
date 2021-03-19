package tzngio

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// The fmt paramter passed to Type.StringOf() must be one of the following
// values, these are used to inform the formatter how containers should be
// encoded and what sort of escaping should be applied to string types.
type OutFmt int

const (
	OutFormatUnescaped = OutFmt(iota)
	OutFormatZNG
	OutFormatZeek
	OutFormatZeekAscii
)

func StringOf(zv zng.Value, out OutFmt, b bool) string {
	switch t := zv.Type.(type) {
	case *zng.TypeArray:
		return StringOfArray(t, zv.Bytes, out, b)
	case *zng.TypeAlias:
		return StringOf(zng.Value{t.Type, zv.Bytes}, out, b)
	case *zng.TypeOfBool:
		b, err := zng.DecodeBool(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		if b {
			return "T"
		}
		return "F"
	case *zng.TypeOfBstring:
		return StringOfBstring(zv.Bytes, out, b)
	case *zng.TypeOfBytes:
		return base64.StdEncoding.EncodeToString(zv.Bytes)
	case *zng.TypeOfDuration:
		return StringOfDuration(t, zv.Bytes, out, b)
	case *zng.TypeEnum:
		return StringOf(zng.Value{zng.TypeUint64, zv.Bytes}, out, false)
	case *zng.TypeOfError:
		return string(zv.Bytes)
	case *zng.TypeOfFloat64:
		d, err := zng.DecodeFloat64(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return strconv.FormatFloat(d, 'f', -1, 64)
	case *zng.TypeOfInt8, *zng.TypeOfInt16, *zng.TypeOfInt32, *zng.TypeOfInt64:
		b, err := zng.DecodeInt(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return strconv.FormatInt(int64(b), 10)
	case *zng.TypeOfUint8, *zng.TypeOfUint16, *zng.TypeOfUint32, *zng.TypeOfUint64:
		b, err := zng.DecodeUint(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return strconv.FormatUint(uint64(b), 10)
	case *zng.TypeOfIP:
		ip, err := zng.DecodeIP(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return ip.String()
	case *zng.TypeMap:
		return StringOfMap(t, zv.Bytes, out, b)
	case *zng.TypeOfNet:
		return StringOfNet(t, zv.Bytes, out, b)
	case *zng.TypeOfNull:
		return "-"
	case *zng.TypeRecord:
		return StringOfRecord(t, zv.Bytes, out, b)
	case *zng.TypeSet:
		return StringOfSet(t, zv.Bytes, out, b)
	case *zng.TypeOfString:
		return StringOfString(t, zv.Bytes, out, b)
	case *zng.TypeOfTime:
		return StringOfTime(t, zv.Bytes, out, b)
	case *zng.TypeOfType:
		return StringOfString(zng.TypeString, zv.Bytes, out, b)
	case *zng.TypeUnion:
		return StringOfUnion(t, zv.Bytes, out, b)
	default:
		return fmt.Sprintf("tzngio.StringOf(): unknown type: %T", t)
	}
}

func StringOfArray(t *zng.TypeArray, zv zcode.Bytes, fmt OutFmt, _ bool) string {
	if len(zv) == 0 && (fmt == OutFormatZeek || fmt == OutFormatZeekAscii) {
		return "(empty)"
	}

	var b strings.Builder
	separator := byte(',')
	if fmt == OutFormatZNG {
		b.WriteByte('[')
		separator = ';'
	}

	first := true
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		if val == nil {
			b.WriteByte('-')
		} else {
			b.WriteString(StringOf(zng.Value{t.Type, val}, fmt, true))
		}
	}

	if fmt == OutFormatZNG {
		if !first {
			b.WriteByte(';')
		}
		b.WriteByte(']')
	}
	return b.String()
}

const hexdigits = "0123456789abcdef"

// Values of type bstring may contain a mix of valid UTF-8 and arbitrary
// binary data.  These are represented in output using the same formatting
// with "\x.." escapes as Zeek.
// In general, valid UTF-8 code points are passed through unmodified,
// though for the ZEEK_ASCII output format, all non-ascii bytes are
// escaped for compatibility with older versions of Zeek.
func StringOfBstring(data zcode.Bytes, fmt OutFmt, inContainer bool) string {
	if bytes.Equal(data, []byte{'-'}) {
		return "\\x2d"
	}

	var out []byte
	var start int
	for i := 0; i < len(data); {
		r, l := utf8.DecodeRune(data[i:])
		if fmt != OutFormatUnescaped && r == '\\' {
			out = append(out, data[start:i]...)
			out = append(out, '\\', '\\')
			i++
			start = i
			continue
		}
		needEscape := r == utf8.RuneError || !unicode.IsPrint(r)
		if !needEscape {
			needEscape = ShouldEscape(r, fmt, i, inContainer)
		}
		if needEscape {
			out = append(out, data[start:i]...)
			// XXX format l chars
			c := data[i]
			out = append(out, '\\', 'x', hexdigits[c>>4], hexdigits[c&0xf])
			i++
			start = i
		} else {
			i += l
		}
	}
	return string(append(out, data[start:len(data)]...))
}

func StringOfDuration(t *zng.TypeOfDuration, zv zcode.Bytes, _ OutFmt, _ bool) string {
	i, err := zng.DecodeDuration(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano Duration. Such values cannot be represented by
	// float64's without loss of the least significant digits of ns,
	return nano.Ts(i).StringFloat()
}

func StringOfMap(t *zng.TypeMap, zv zcode.Bytes, fmt OutFmt, _ bool) string {
	var b strings.Builder
	it := zv.Iter()
	b.WriteByte('[')
	for !it.Done() {
		val, container, err := it.Next()
		if err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		b.WriteString(StringOf(zng.Value{t.KeyType, val}, fmt, true))
		if !container {
			b.WriteByte(';')
		}
		val, container, err = it.Next()
		if err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		b.WriteString(StringOf(zng.Value{t.ValType, val}, fmt, true))
		if !container {
			b.WriteByte(';')
		}
	}
	b.WriteByte(']')
	return b.String()
}

func StringOfNet(t *zng.TypeOfNet, zv zcode.Bytes, _ OutFmt, _ bool) string {
	s, err := zng.DecodeNet(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	ipnet := net.IPNet(*s)
	return ipnet.String()
}

func StringOfRecord(t *zng.TypeRecord, zv zcode.Bytes, fmt OutFmt, _ bool) string {
	var b strings.Builder
	separator := byte(',')
	if fmt == OutFormatZNG {
		b.WriteByte('[')
		separator = ';'
	}

	first := true
	it := zv.Iter()
	for _, col := range t.Columns {
		val, _, err := it.Next()
		if err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		if val == nil {
			b.WriteByte('-')
		} else {
			b.WriteString(StringOf(zng.Value{col.Type, val}, fmt, false))
		}
	}

	if fmt == OutFormatZNG {
		if !first {
			b.WriteByte(';')
		}
		b.WriteByte(']')
	}
	return b.String()
}

func StringOfSet(t *zng.TypeSet, zv zcode.Bytes, fmt OutFmt, _ bool) string {
	if len(zv) == 0 && (fmt == OutFormatZeek || fmt == OutFormatZeekAscii) {
		return "(empty)"
	}

	var b strings.Builder
	separator := byte(',')
	if fmt == OutFormatZNG {
		b.WriteByte('[')
		separator = ';'
	}

	first := true
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		b.WriteString(StringOf(zng.Value{t.Type, val}, fmt, true))
	}

	if fmt == OutFormatZNG {
		if !first {
			b.WriteByte(';')
		}
		b.WriteByte(']')
	}
	return b.String()
}

func StringOfString(t *zng.TypeOfString, zv zcode.Bytes, fmt OutFmt, inContainer bool) string {
	if fmt != OutFormatUnescaped && bytes.Equal(zv, []byte{'-'}) {
		return "\\u002d"
	}

	var out []byte
	var start int
	for i := 0; i < len(zv); {
		r, l := utf8.DecodeRune(zv[i:])
		if fmt != OutFormatUnescaped && r == '\\' {
			out = append(out, zv[start:i]...)
			out = append(out, '\\', '\\')
			i++
			start = i
			continue
		}
		if !unicode.IsPrint(r) || ShouldEscape(r, fmt, i, inContainer) {
			out = append(out, zv[start:i]...)
			out = append(out, uescape(r)...)
			i += l
			start = i
		} else {
			i += l
		}
	}
	return string(append(out, zv[start:len(zv)]...))
}

func uescape(r rune) []byte {
	code := strconv.FormatInt(int64(r), 16)
	var s string
	if len(code) == 4 {
		s = fmt.Sprintf("\\u%s", code)
	} else {
		s = fmt.Sprintf("\\u{%s}", code)
	}
	return []byte(s)
}

func StringOfTime(t *zng.TypeOfTime, zv zcode.Bytes, _ OutFmt, _ bool) string {
	ts, err := zng.DecodeTime(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano.Ts.  Such values cannot be representd by
	// float64's without loss of the least significant digits of ns,
	return ts.StringFloat()
}

func StringOfUnion(t *zng.TypeUnion, zv zcode.Bytes, ofmt OutFmt, _ bool) string {
	typ, index, iv, err := t.SplitZng(zv)
	if err != nil {
		// this follows set and record StringOfs. Like there, XXX.
		return "ERR"
	}

	s := strconv.FormatInt(index, 10) + ":"
	return s + StringOf(zng.Value{typ, iv}, ofmt, false)
}
