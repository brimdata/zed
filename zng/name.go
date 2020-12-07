package zng

import (
	"strings"
	"unicode"
)

func IdChar(c rune) bool {
	return unicode.IsLetter(c) || c == '_' || c == '$'
}

func IsIdentifier(s string) bool {
	first := true
	for _, c := range s {
		if !IdChar(c) && (first || !unicode.IsDigit(c)) {
			return false
		}
		first = false
	}
	return true
}

func FormatName(name string) string {
	if IsIdentifier(name) {
		return name
	}
	var b strings.Builder
	b.WriteRune('[')
	b.WriteString(TypeString.StringOf(EncodeString(name), OutFormatZNG, false))
	b.WriteRune(']')
	return b.String()
}
