package zed

import (
	"strings"
	"unicode/utf8"
)

func QuotedName(name string) string {
	if !IsIdentifier(name) {
		name = QuotedString([]byte(name), false)
	}
	return name
}

const hexdigits = "0123456789abcdef"

func QuotedString(s []byte, _ bool) string {
	var b strings.Builder
	b.WriteByte('"')
	for k := 0; k < len(s); {
		if c := s[k]; c < utf8.RuneSelf {
			if safeSet[c] {
				b.WriteByte(c)
				k++
				continue
			}
			b.WriteByte('\\')
			switch c {
			case '\\', '"':
				b.WriteByte(c)
			case '\b':
				b.WriteByte('b')
			case '\f':
				b.WriteByte('f')
			case '\n':
				b.WriteByte('n')
			case '\r':
				b.WriteByte('r')
			case '\t':
				b.WriteByte('t')
			default:
				// ASCII control codes other than above
				b.WriteString(`u00`)
				b.WriteByte(hexdigits[c>>4])
				b.WriteByte(hexdigits[c&0xF])
			}
			k++
			continue
		}
		r, size := utf8.DecodeRune(s[k:])
		if r == utf8.RuneError && size == 1 {
			//XXX panic instead?  Caller should ensure
			// string is valid uft8 (e.g., verified ZNG)
			b.WriteString(`\ufffd`)
			k += size
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if r == '\u2028' || r == '\u2029' {
			b.WriteString(`\u202`)
			b.WriteByte(hexdigits[r&0xF])
			k += size
			continue
		}
		b.WriteRune(r)
		k += size
	}
	b.WriteByte('"')
	return b.String()
}

func esc(r rune) byte {
	switch r {
	case '\\':
		return '\\'
	case '"':
		return '"'
	case '\b':
		return 'b'
	case '\f':
		return 'f'
	case '\n':
		return 'n'
	case '\r':
		return 'r'
	case '\t':
		return 't'
	}
	return 0
}

func Unhex(b byte) byte {
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

//XXX from golang json package

// safeSet holds the value true if the ASCII character with the given array
// position can be represented inside a JSON string without any further
// escaping.
//
// All values are true except for the ASCII control characters (0-31), the
// double quote ("), and the backslash character ("\").
var safeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      true,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      true,
	'=':      true,
	'>':      true,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}
