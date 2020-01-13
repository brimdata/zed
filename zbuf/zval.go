package zbuf

import (
	"bytes"
	"strings"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

// ZvalFromZeekString returns the zval for the Zeek UTF-8 value described by typ
// and val.
func ZvalFromZeekString(typ zng.Type, val string) ([]byte, error) {
	it := zcode.Iter(appendZvalFromZeek(nil, typ, []byte(val)))
	v, _, err := it.Next()
	return v, err
}

// appendZvalFromZeek appends to dst the zval for the Zeek UTF-8 value described
// by typ and val.
func appendZvalFromZeek(dst zcode.Bytes, typ zng.Type, val []byte) zcode.Bytes {
	const empty = "(empty)"
	const setSeparator = ','
	const unset = '-'
	switch typ.(type) {
	case *zng.TypeSet, *zng.TypeVector:
		if bytes.Equal(val, []byte{unset}) {
			return zcode.AppendContainer(dst, nil)
		}
		inner := zng.InnerType(typ)
		zv := make(zcode.Bytes, 0)
		if !bytes.Equal(val, []byte(empty)) {
			for _, v := range bytes.Split(val, []byte{setSeparator}) {
				body, _ := inner.Parse(zng.Unescape(v))
				zv = zcode.AppendSimple(zv, body)
			}
		}
		return zcode.AppendContainer(dst, zv)
	default:
		if bytes.Equal(val, []byte{unset}) {
			return zcode.AppendSimple(dst, nil)
		}
		body, _ := typ.Parse(zng.Unescape(val))
		return zcode.AppendSimple(dst, body)
	}
}

func escape(s string, utf8 bool) string {
	if utf8 {
		return zng.EscapeUTF8([]byte(s))
	}
	return zng.Escape([]byte(s))
}

// ZvalToZeekString returns a Zeek ASCII string representing the zval described
// by typ and val.
func ZvalToZeekString(typ zng.Type, zv zcode.Bytes, isContainer bool, utf8 bool) string {
	if zv == nil {
		return "-"
	}
	var s string
	switch typ.(type) {
	case *zng.TypeSet, *zng.TypeVector:
		inner := zng.InnerType(typ)
		if len(zv) == 0 {
			return "(empty)"
		}
		// XXX handle one value that equals "(empty)"
		var b strings.Builder
		it := zcode.Iter(zv)
		for {
			v, _, err := it.Next()
			if err != nil {
				return "error in ZvalToZeekString"
			}
			val, err := inner.New(v)
			if err != nil {
				return "error in ZvalToZeekString"
			}
			fld := escape(val.String(), utf8)
			// Escape the set separator after ZeekEscape.
			_, _ = b.WriteString(strings.ReplaceAll(fld, ",", "\\x2c"))
			if it.Done() {
				break
			}
			_ = b.WriteByte(',')
		}
		s = b.String()
	default:
		val, err := typ.New(zv)
		if err != nil {
			return "error in ZvalToZeekString"
		}
		s = escape(val.String(), utf8)
	}
	if s == "-" {
		return "\\x2d"
	}
	return s
}
