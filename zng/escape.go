package zng

import (
	"bytes"
)

// ShouldEscape determines if the given code point at the given position
// in a value should be escaped for the given output format.  This function
// does not account for unprintable characters, its main purpose is to
// centralize the logic about which characters are syntatically significant
// in each output format and hence must be escaped.
func ShouldEscape(r rune, fmt OutFmt, pos int) bool {
	if fmt != OUT_FORMAT_UNESCAPED && r == '\\' {
		return true
	}

	if fmt == OUT_FORMAT_ZNG && (r == '\\' || r == ';' || (pos == 0 && r == '[')) {
		return true
	}

	if (fmt == OUT_FORMAT_ZEEK || fmt == OUT_FORMAT_ZEEK_ASCII) && (r == '\\' || r == '\t' || r == ',') {
		return true
	}

	if fmt == OUT_FORMAT_ZEEK_ASCII && r > 0x7f {
		return true
	}
	return false
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
