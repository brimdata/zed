package tzngio

import (
	"regexp"

	"github.com/brimdata/zed"
)

// ShouldEscape determines if the given code point at the given position
// in a value should be escaped for the given output format.  This function
// does not account for unprintable characters, its main purpose is to
// centralize the logic about which characters are syntatically significant
// in each output format and hence must be escaped.  The inContainer parameter
// specifies whether this values is inside a set or vector (which is needed
// to correctly implement  zeek log escaping rules).
func ShouldEscape(r rune, fmt OutFmt, pos int, inContainer bool) bool {
	if fmt != OutFormatUnescaped && r == '\\' {
		return true
	}

	if fmt == OutFormatZNG && (r == ';' || (pos == 0 && (r == '[' || r == ']'))) {
		return true
	}

	if (fmt == OutFormatZeek || fmt == OutFormatZeekAscii) && (r == '\t' || (r == ',' && inContainer)) {
		return true
	}

	if fmt == OutFormatZeekAscii && r > 0x7f {
		return true
	}
	return false
}

func replaceStringEscape(in []byte) []byte {
	var r rune
	i := 2
	if in[i] == '{' {
		i++
	}
	for ; i < len(in) && in[i] != '}'; i++ {
		r <<= 4
		r |= rune(zed.Unhex(in[i]))
	}
	return []byte(string(r))
}

var pattern = regexp.MustCompile(`\\u([0-9A-Fa-f]{4}|\{[0-9A-Fa-f]{1,6}\})`)

// UnescapeString replaces all the escaped characters defined in the
// for the zng spec for the string type with their unescaped equivalents.
func UnescapeString(data []byte) []byte {
	r := pattern.ReplaceAllFunc(data, replaceStringEscape)
	// ReplaceAllFunc() returns nil when data is an empty string but the
	// difference is meaningful inside zng...
	if r == nil {
		return data
	}
	return r
}
