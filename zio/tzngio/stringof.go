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

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
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

func StringOf(zv zed.Value, out OutFmt, b bool) string {
	switch t := zv.Type.(type) {
	case *zed.TypeArray:
		return StringOfArray(t, zv.Bytes, out, b)
	case *zed.TypeAlias:
		return StringOf(zed.Value{t.Type, zv.Bytes}, out, b)
	case *zed.TypeOfBool:
		b, err := zed.DecodeBool(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		if b {
			return "T"
		}
		return "F"
	case *zed.TypeOfBstring:
		return StringOfBstring(zv.Bytes, out, b)
	case *zed.TypeOfBytes:
		return base64.StdEncoding.EncodeToString(zv.Bytes)
	case *zed.TypeOfDuration:
		return StringOfDuration(t, zv.Bytes, out, b)
	case *zed.TypeEnum:
		return StringOf(zed.Value{zed.TypeUint64, zv.Bytes}, out, false)
	case *zed.TypeOfError:
		return string(zv.Bytes)
	case *zed.TypeOfFloat32:
		v, err := zed.DecodeFloat32(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case *zed.TypeOfFloat64:
		d, err := zed.DecodeFloat64(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return strconv.FormatFloat(d, 'f', -1, 64)
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64:
		b, err := zed.DecodeInt(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return strconv.FormatInt(int64(b), 10)
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32, *zed.TypeOfUint64:
		b, err := zed.DecodeUint(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return strconv.FormatUint(uint64(b), 10)
	case *zed.TypeOfIP:
		ip, err := zed.DecodeIP(zv.Bytes)
		if err != nil {
			return badZng(err, t, zv.Bytes)
		}
		return ip.String()
	case *zed.TypeMap:
		return StringOfMap(t, zv.Bytes, out, b)
	case *zed.TypeOfNet:
		return StringOfNet(t, zv.Bytes, out, b)
	case *zed.TypeOfNull:
		return "-"
	case *zed.TypeRecord:
		return StringOfRecord(t, zv.Bytes, out, b)
	case *zed.TypeSet:
		return StringOfSet(t, zv.Bytes, out, b)
	case *zed.TypeOfString:
		return StringOfString(t, zv.Bytes, out, b)
	case *zed.TypeOfTime:
		return StringOfTime(t, zv.Bytes, out, b)
	case *zed.TypeOfType:
		s, _ := zson.FormatValue(zv)
		return s
	case *zed.TypeUnion:
		return StringOfUnion(t, zv.Bytes, out, b)
	default:
		return fmt.Sprintf("tzngio.StringOf(): unknown type: %T", t)
	}
}

func StringOfArray(t *zed.TypeArray, zv zcode.Bytes, fmt OutFmt, _ bool) string {
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
			b.WriteString(StringOf(zed.Value{t.Type, val}, fmt, true))
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
	return string(append(out, data[start:]...))
}

func StringOfDuration(t *zed.TypeOfDuration, zv zcode.Bytes, _ OutFmt, _ bool) string {
	i, err := zed.DecodeDuration(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano Duration. Such values cannot be represented by
	// float64's without loss of the least significant digits of ns,
	return nano.Ts(i).StringFloat()
}

func StringOfMap(t *zed.TypeMap, zv zcode.Bytes, fmt OutFmt, _ bool) string {
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
		b.WriteString(StringOf(zed.Value{t.KeyType, val}, fmt, true))
		if !container {
			b.WriteByte(';')
		}
		val, container, err = it.Next()
		if err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		b.WriteString(StringOf(zed.Value{t.ValType, val}, fmt, true))
		if !container {
			b.WriteByte(';')
		}
	}
	b.WriteByte(']')
	return b.String()
}

func StringOfNet(t *zed.TypeOfNet, zv zcode.Bytes, _ OutFmt, _ bool) string {
	s, err := zed.DecodeNet(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	ipnet := net.IPNet(*s)
	return ipnet.String()
}

func StringOfRecord(t *zed.TypeRecord, zv zcode.Bytes, fmt OutFmt, _ bool) string {
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
			b.WriteString(StringOf(zed.Value{col.Type, val}, fmt, false))
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

func StringOfSet(t *zed.TypeSet, zv zcode.Bytes, fmt OutFmt, _ bool) string {
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
		b.WriteString(StringOf(zed.Value{t.Type, val}, fmt, true))
	}

	if fmt == OutFormatZNG {
		if !first {
			b.WriteByte(';')
		}
		b.WriteByte(']')
	}
	return b.String()
}

func StringOfString(t *zed.TypeOfString, zv zcode.Bytes, fmt OutFmt, inContainer bool) string {
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
	return string(append(out, zv[start:]...))
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

func StringOfTime(t *zed.TypeOfTime, zv zcode.Bytes, _ OutFmt, _ bool) string {
	ts, err := zed.DecodeTime(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano.Ts.  Such values cannot be representd by
	// float64's without loss of the least significant digits of ns,
	return ts.StringFloat()
}

func StringOfUnion(t *zed.TypeUnion, zv zcode.Bytes, ofmt OutFmt, _ bool) string {
	typ, selector, iv, err := t.SplitZng(zv)
	if err != nil {
		// this follows set and record StringOfs. Like there, XXX.
		return "ERR"
	}

	s := strconv.FormatInt(selector, 10) + ":"
	return s + StringOf(zed.Value{typ, iv}, ofmt, false)
}
