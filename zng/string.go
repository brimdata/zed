package zng

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/mccanne/zq/zcode"
	"golang.org/x/text/unicode/norm"
)

type TypeOfString struct{}

func NewString(s string) Value {
	return Value{TypeString, EncodeString(s)}
}

func EncodeString(s string) zcode.Bytes {
	return zcode.Bytes(s)
}

func DecodeString(zv zcode.Bytes) (string, error) {
	if zv == nil {
		return "", ErrUnset
	}
	return string(zv), nil
}

func (t *TypeOfString) Parse(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(Unescape(in))
	return normalized, nil
}

func (t *TypeOfString) ID() int {
	return IdString
}

func (t *TypeOfString) String() string {
	return "string"
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

func (t *TypeOfString) StringOf(zv zcode.Bytes, ofmt OutFmt) string {
	if bytes.Equal(zv, []byte{'-'}) {
		return "\\u002d"
	}

	var out []byte
	var start int
	for i := 0; i < len(zv); {
		r, l := utf8.DecodeRune(zv[i:])
		if ofmt != OUT_FORMAT_UNESCAPED && r == '\\' {
			out = append(out, zv[start:i]...)
			out = append(out, '\\', '\\')
			i++
			start = i
			continue
		}
		if !unicode.IsPrint(r) || ShouldEscape(r, ofmt, i) {
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

func (t *TypeOfString) Marshal(zv zcode.Bytes) (interface{}, error) {
	return t.StringOf(zv, OUT_FORMAT_UNESCAPED), nil
}
