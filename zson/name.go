package zson

import (
	"strconv"
	"unicode"
)

func IsIdentifier(s string) bool {
	if s == "" {
		return false
	}
	first := true
	for _, c := range s {
		if !idChar(c) && (first || !unicode.IsDigit(c)) {
			return false
		}
		first = false
	}
	return true
}

func idChar(c rune) bool {
	return unicode.IsLetter(c) || c == '_' || c == '$'
}

// IsTypeName returns true iff s is a valid zson typedef name (exclusive
// of integer names for locally-scoped typedefs).
func IsTypeName(s string) bool {
	for _, c := range s {
		if !typeChar(c) {
			return false
		}
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err != nil
}

func typeChar(c rune) bool {
	return idChar(c) || unicode.IsDigit(c) || c == '/' || c == '.'
}
