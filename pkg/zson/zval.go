package zson

import (
	"bytes"
	"strings"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
)

// ZvalFromZeekString returns the zval for the Zeek UTF-8 value described by typ
// and val.
func ZvalFromZeekString(typ zeek.Type, val string) ([]byte, error) {
	it := zval.Iter(appendZvalFromZeek(nil, typ, []byte(val)))
	v, _, err := it.Next()
	return v, err
}

// appendZvalFromZeek appends to dst the zval for the Zeek UTF-8 value described
// by typ and val.
func appendZvalFromZeek(dst zval.Encoding, typ zeek.Type, val []byte) zval.Encoding {
	const empty = "(empty)"
	const setSeparator = ','
	const unset = '-'
	switch typ.(type) {
	case *zeek.TypeSet, *zeek.TypeVector:
		if bytes.Equal(val, []byte{unset}) {
			return zval.AppendContainer(dst, nil)
		}
		inner := zeek.InnerType(typ)
		zv := make(zval.Encoding, 0)
		if !bytes.Equal(val, []byte(empty)) {
			for _, v := range bytes.Split(val, []byte{setSeparator}) {
				body, _ := inner.Parse(zeek.Unescape(v))
				zv = zval.AppendValue(zv, body)
			}
		}
		return zval.Append(dst, zv, true)
	default:
		if bytes.Equal(val, []byte{unset}) {
			return zval.AppendValue(dst, nil)
		}
		body, _ := typ.Parse(zeek.Unescape(val))
		return zval.AppendValue(dst, body)
	}
}

func escape(s string, utf8 bool) string {
	if utf8 {
		return zeek.EscapeUTF8([]byte(s))
	}
	return zeek.Escape([]byte(s))
}

// ZvalToZeekString returns a Zeek ASCII string representing the zval described
// by typ and val.
func ZvalToZeekString(typ zeek.Type, zv zval.Encoding, isContainer bool, utf8 bool) string {
	if zv == nil {
		return "-"
	}
	var s string
	switch typ.(type) {
	case *zeek.TypeSet, *zeek.TypeVector:
		inner := zeek.InnerType(typ)
		if len(zv) == 0 {
			return "(empty)"
		}
		// XXX handle one value that equals "(empty)"
		var b strings.Builder
		it := zval.Iter(zv)
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
