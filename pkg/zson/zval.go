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
	return it.Next()
}

// appendZvalFromZeek appends to dst the zval for the Zeek UTF-8 value described
// by typ and val.
func appendZvalFromZeek(dst []byte, typ zeek.Type, val []byte) []byte {
	const empty = "(empty)"
	const setSeparator = ','
	const unset = '-'
	if bytes.Equal(val, []byte{unset}) {
		// An unset field is encoded as a zero-length field.
		return zval.AppendUvarint(dst, 0)
	}
	switch typ.(type) {
	case *zeek.TypeSet, *zeek.TypeVector:
		var vals [][]byte
		if !bytes.Equal(val, []byte(empty)) {
			for _, v := range bytes.Split(val, []byte{setSeparator}) {
				vals = append(vals, zeek.Unescape(v))
			}
		}
		return zval.AppendContainer(dst, vals)
	default:
		return zval.AppendValue(dst, zeek.Unescape(val))
	}
}

// ZvalToZeekString returns a Zeek ASCII string representing the zval described
// by typ and val.
func ZvalToZeekString(typ zeek.Type, val []byte) string {
	if val == nil {
		return "-"
	}
	var s string
	switch typ.(type) {
	case *zeek.TypeSet, *zeek.TypeVector:
		if len(val) == 0 {
			return "(empty)"
		}
		// XXX handle one value that equals "(empty)"
		var b strings.Builder
		it := zval.Iter(val)
		for {
			v, err := it.Next()
			if err != nil {
				return zeek.Escape(val)
			}
			// Escape the set separator after ZeekEscape.n
			_, _ = b.WriteString(strings.ReplaceAll(zeek.Escape(v), ",", "\\x2c"))
			if it.Done() {
				break
			}
			_ = b.WriteByte(',')
		}
		s = b.String()
	default:
		s = zeek.Escape(val)
	}
	if s == "-" {
		return "\\x2d"
	}
	return s
}
