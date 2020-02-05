package zng

import (
	"bytes"
	"github.com/mccanne/zq/zcode"
	"golang.org/x/text/unicode/norm"
	"unicode"
	"unicode/utf8"
)

type TypeOfBstring struct{}

func NewBstring(s string) Value {
	return Value{TypeBstring, EncodeBstring(s)}
}

func EncodeBstring(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeBstring(zv zcode.Bytes) (string, error) {
	if zv == nil {
		return "", ErrUnset
	}
	return string(zv), nil
}

func (t *TypeOfBstring) Parse(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(UnescapeBstring(in))
	return normalized, nil
}

func (t *TypeOfBstring) ID() int {
	return IdBstring
}

func (t *TypeOfBstring) String() string {
	return "bstring"
}

func (t *TypeOfBstring) Marshal(zv zcode.Bytes) (interface{}, error) {
	return t.StringOf(zv, OutFormatUnescaped, false), nil
}

const hexdigits = "0123456789abcdef"

// Values of type bstring may contain a mix of valid UTF-8 and arbitrary
// binary data.  These are represented in output using the same formatting
// with "\x.." escapes as Zeek.
// In general, valid UTF-8 code points are passed through unmodified,
// though for the ZEEK_ASCII output format, all non-ascii bytes are
// escaped for compatibility with older versions of Zeek.
func (t *TypeOfBstring) StringOf(data zcode.Bytes, fmt OutFmt, inContainer bool) string {
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
