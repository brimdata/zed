package zeekio

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

func badZNG(err error, t zed.Type, zv zcode.Bytes) string {
	return fmt.Sprintf("<ZNG-ERR type %s [%s]: %s>", t, zv, err)
}

func formatAny(zv zed.Value, inContainer bool) string {
	switch t := zv.Type.(type) {
	case *zed.TypeArray:
		return formatArray(t, zv.Bytes)
	case *zed.TypeAlias:
		return formatAny(zed.Value{t.Type, zv.Bytes}, inContainer)
	case *zed.TypeOfBool:
		b, err := zed.DecodeBool(zv.Bytes)
		if err != nil {
			return badZNG(err, t, zv.Bytes)
		}
		if b {
			return "T"
		}
		return "F"
	case *zed.TypeOfBytes:
		return base64.StdEncoding.EncodeToString(zv.Bytes)
	case *zed.TypeOfDuration:
		return formatDuration(t, zv.Bytes)
	case *zed.TypeEnum:
		return formatAny(zed.Value{zed.TypeUint64, zv.Bytes}, false)
	case *zed.TypeOfError:
		return string(zv.Bytes)
	case *zed.TypeOfFloat32:
		v, err := zed.DecodeFloat32(zv.Bytes)
		if err != nil {
			return badZNG(err, t, zv.Bytes)
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case *zed.TypeOfFloat64:
		d, err := zed.DecodeFloat64(zv.Bytes)
		if err != nil {
			return badZNG(err, t, zv.Bytes)
		}
		return strconv.FormatFloat(d, 'f', -1, 64)
	case *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64:
		b, err := zed.DecodeInt(zv.Bytes)
		if err != nil {
			return badZNG(err, t, zv.Bytes)
		}
		return strconv.FormatInt(int64(b), 10)
	case *zed.TypeOfUint8, *zed.TypeOfUint16, *zed.TypeOfUint32, *zed.TypeOfUint64:
		b, err := zed.DecodeUint(zv.Bytes)
		if err != nil {
			return badZNG(err, t, zv.Bytes)
		}
		return strconv.FormatUint(uint64(b), 10)
	case *zed.TypeOfIP:
		ip, err := zed.DecodeIP(zv.Bytes)
		if err != nil {
			return badZNG(err, t, zv.Bytes)
		}
		return ip.String()
	case *zed.TypeMap:
		return formatMap(t, zv.Bytes)
	case *zed.TypeOfNet:
		return formatNet(t, zv.Bytes)
	case *zed.TypeOfNull:
		return "-"
	case *zed.TypeRecord:
		return formatRecord(t, zv.Bytes)
	case *zed.TypeSet:
		return formatSet(t, zv.Bytes)
	case *zed.TypeOfString:
		return formatString(t, zv.Bytes, inContainer)
	case *zed.TypeOfTime:
		return formatTime(t, zv.Bytes)
	case *zed.TypeOfType:
		s, _ := zson.FormatValue(zv)
		return s
	case *zed.TypeUnion:
		return formatUnion(t, zv.Bytes)
	default:
		return fmt.Sprintf("tzngio.StringOf(): unknown type: %T", t)
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
		if val, _ := it.Next(); val == nil {
			b.WriteByte('-')
		} else {
			b.WriteString(formatAny(zed.Value{t.Type, val}, true))
		}
	}
	return b.String()
}

func formatDuration(t *zed.TypeOfDuration, zv zcode.Bytes) string {
	i, err := zed.DecodeDuration(zv)
	if err != nil {
		return badZNG(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano Duration. Such values cannot be represented by
	// float64's without loss of the least significant digits of ns,
	return nano.Ts(i).StringFloat()
}

func formatMap(t *zed.TypeMap, zv zcode.Bytes) string {
	var b strings.Builder
	it := zv.Iter()
	b.WriteByte('[')
	for !it.Done() {
		val, container := it.Next()
		b.WriteString(formatAny(zed.Value{t.KeyType, val}, true))
		if !container {
			b.WriteByte(';')
		}
		val, container = it.Next()
		b.WriteString(formatAny(zed.Value{t.ValType, val}, true))
		if !container {
			b.WriteByte(';')
		}
	}
	b.WriteByte(']')
	return b.String()
}

func formatNet(t *zed.TypeOfNet, zv zcode.Bytes) string {
	s, err := zed.DecodeNet(zv)
	if err != nil {
		return badZNG(err, t, zv)
	}
	ipnet := net.IPNet(*s)
	return ipnet.String()
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
		if val, _ := it.Next(); val == nil {
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
		val, _ := it.Next()
		b.WriteString(formatAny(zed.Value{t.Type, val}, true))
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

func formatTime(t *zed.TypeOfTime, zv zcode.Bytes) string {
	ts, err := zed.DecodeTime(zv)
	if err != nil {
		return badZNG(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano.Ts.  Such values cannot be representd by
	// float64's without loss of the least significant digits of ns,
	return ts.StringFloat()
}

func formatUnion(t *zed.TypeUnion, zv zcode.Bytes) string {
	typ, selector, iv, err := t.SplitZNG(zv)
	if err != nil {
		// this follows set and record StringOfs. Like there, XXX.
		return "ERR"
	}

	s := strconv.FormatInt(selector, 10) + ":"
	return s + formatAny(zed.Value{typ, iv}, false)
}

func FormatValue(v zed.Value) string {
	if v.Bytes == nil {
		return "-"
	}
	return formatAny(v, false)
}
