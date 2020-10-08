package zng

import (
	"strings"
	"unicode"
)

func idChar(c rune) bool {
	return unicode.IsLetter(c) || c == '_' || c == '$'
}

func NameIsId(s string) bool {
	first := true
	for _, c := range s {
		if !idChar(c) && (first || !unicode.IsDigit(c)) {
			return false
		}
		first = false
	}
	return true
}

func FormatName(name string) string {
	if NameIsId(name) {
		return name
	}
	var b strings.Builder
	b.WriteRune('[')
	b.WriteString(TypeString.StringOf(EncodeString(name), OutFormatZNG, false))
	b.WriteRune(']')
	return b.String()
}
