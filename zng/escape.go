package zng

import (
	"bytes"
	"unicode"
	"unicode/utf8"
	"unsafe"
)

const hexdigits = "0123456789abcdef"

// Escape returns a representation of data with \ replaced by \\ and with all
// bytes outside the range from 0x20 through 0x7e replaced by a \xhh sequence.
// This is string escaping scheme implemented by Zeek's ASCII log writer.
func Escape(data []byte) string {
	var buf []byte
	for _, c := range data {
		switch {
		case c == '\\':
			buf = append(buf, c, c)
		case c < 0x20 || 0x7e < c:
			buf = append(buf, '\\', 'x', hexdigits[c>>4], hexdigits[c&0xf])
		default:
			buf = append(buf, c)
		}
	}
	return string(buf)
}

// EscapeUTF8 does the same non-standard formatting of mixed-binary strings
// that zeek does.  There is no way to disambiguate between random binary data
// and a deliberate utf-8 string so it's left to the "presenation layer" to
// decide how to format the data.  This is not an issue for the ZNG types
// "string" (which must always be valid UTF-8) and "bytes" (which is defined
// to be an anonymous buffer of bytes and hence not treated as text).  But
// data coming from legacy Zeek logs maps to the "bstring" type which is
// treated as UTF-8 when possible but which may not always contain valid
// UTF-8.
//
// For comparison, the enable_utf_8 option in the Zeek ascii writer governs
// whether characters that are not printable ascii are escaped or are
// interpreted as UTF-8:
// https://docs.zeek.org/en/stable/scripts/base/frameworks/logging/writers/ascii.zeek.html#id-LogAscii::enable_utf_8
// The Zeek JSON writer assumes UTF-8 encoding and does not escape valid UTF-8.
//
// This function is used to escape non-printable characters with \x syntax.
// If invalidOnly is false, all non-printable characters are escaped.  If
// invalidOnly is true, only bytes that cannot be decoded as UTF-8 are escaped.
func EscapeUTF8(data []byte, invalidOnly bool) string {
	var out []byte
	var start int
	for i, r := range *(*string)(unsafe.Pointer(&data)) {
		if !unicode.IsPrint(r) && (!invalidOnly || r == utf8.RuneError) {
			out = append(out, data[start:i]...)
			c := data[i]
			out = append(out, '\\', 'x', hexdigits[c>>4], hexdigits[c&0xf])
			start = i + 1
		}
	}
	return string(append(out, data[start:len(data)]...))
}

// Unescape is the inverse of Escape.
func Unescape(data []byte) []byte {
	if bytes.IndexByte(data, '\\') < 0 {
		return data
	}
	var buf []byte
	i := 0
	for i < len(data) {
		c := data[i]
		if c == '\\' && len(data[i:]) >= 2 {
			var n int
			c, n = ParseEscape(data[i:])
			i += n
		} else {
			i++
		}
		buf = append(buf, c)
	}
	return buf
}

func ParseEscape(data []byte) (byte, int) {
	if len(data) >= 4 && data[1] == 'x' {
		v1 := unhex(data[2])
		v2 := unhex(data[3])
		if v1 <= 0xf || v2 <= 0xf {
			return v1<<4 | v2, 4
		}
	}
	return data[1], 2
}

func unhex(b byte) byte {
	switch {
	case '0' <= b && b <= '9':
		return b - '0'
	case 'a' <= b && b <= 'f':
		return b - 'a' + 10
	case 'A' <= b && b <= 'F':
		return b - 'A' + 10
	}
	return 255
}
