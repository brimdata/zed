package zson

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/brimsec/zq/zng"
)

const primitiveRE = `^(([0-9a-fA-Fx_\$\-\+:eEnumsh./TZµ]+)|true|false|null)`
const indentationRE = `\n\s*`

type Lexer struct {
	buffer      []byte
	cursor      []byte
	primitive   *regexp.Regexp
	indentation *regexp.Regexp
}

func NewLexer(r io.Reader) (*Lexer, error) {
	// XXX Slurping in the entire file faciliated the implementation of
	// the scanner logic here.  Now that the design is fleshed out, we need
	// tweak things here to read from the input as a stream.  See issue #1802.
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	primitive := regexp.MustCompile(primitiveRE)
	primitive.Longest()
	indentation := regexp.MustCompile(indentationRE)
	indentation.Longest()
	return &Lexer{
		buffer:      b,
		cursor:      b,
		primitive:   primitive,
		indentation: indentation,
	}, nil
}

func (l *Lexer) skip(n int) {
	l.cursor = l.cursor[n:]
}

func (l *Lexer) peek() (byte, error) {
	if len(l.cursor) > 1 {
		return l.cursor[0], nil
	}
	return 0, io.EOF
}

func (l *Lexer) match(b byte) (bool, error) {
	if err := l.skipSpace(); err != nil {
		return false, err
	}
	return l.matchTight(b)
}

func (l *Lexer) matchTight(b byte) (bool, error) {
	if len(l.cursor) == 0 {
		return false, io.EOF
	}
	if b == l.cursor[0] {
		l.skip(1)
		return true, nil
	}
	return false, nil
}

func (l *Lexer) matchBytes(b []byte) (bool, error) {
	if err := l.skipSpace(); err != nil {
		return false, err
	}
	return l.matchBytesTight(b)
}

func (l *Lexer) matchBytesTight(b []byte) (bool, error) {
	if len(l.cursor) == 0 {
		return false, io.EOF
	}
	n := len(b)
	if n > len(l.cursor) {
		return false, io.EOF
	}
	ok := bytes.Equal(b, l.cursor[:n])
	if ok {
		l.skip(n)
	}
	return ok, nil
}

func (l *Lexer) skipTo(b byte) ([]byte, error) {
	for off := 0; off < len(l.cursor); off++ {
		c := l.cursor[off]
		if c == b {
			return l.cursor[:off+1], nil
		}
	}
	return nil, nil
}

func (l *Lexer) readByte() (byte, error) {
	if len(l.cursor) == 0 {
		return 0, io.EOF
	}
	b := l.cursor[0]
	l.cursor = l.cursor[1:]
	return b, nil
}

func (l *Lexer) peekRune() (rune, int, error) {
	r, n := utf8.DecodeRune(l.cursor)
	return r, n, nil
}

func (l *Lexer) peekRuneAt(n int) (rune, int, error) {
	if len(l.cursor) < n {
		return 0, 0, io.EOF
	}
	r, n := utf8.DecodeRune(l.cursor[n:])
	return r, n, nil
}

var slashCommentStart = []byte("//")
var starCommentStart = []byte("/*")

func (l *Lexer) skipSpace() error {
	for {
		r, n, err := l.peekRune()
		if err != nil {
			return err
		}
		if unicode.IsSpace(r) {
			l.skip(n)
			continue
		}
		if r == '/' {
			ok, err := l.matchBytesTight(slashCommentStart)
			if err != nil {
				return err
			}
			if ok {
				if err := l.skipLine(); err != nil {
					return err
				}
				continue
			}
			ok, err = l.matchBytesTight(starCommentStart)
			if err != nil {
				return err
			}
			if ok {
				if err := l.skipMultiLine(); err != nil {
					return err
				}
				continue
			}
		}
		return nil
	}
}

func isNewline(r rune) bool {
	// See http://www.unicode.org/versions/Unicode13.0.0/ch05.pdf#G10213
	switch r {
	case 0x000A, 0x000B, 0x000C, 0x000D, 0x0085, 0x2028, 0x2029:
		return true
	}
	return false
}

func (l *Lexer) skipLine() error {
	for {
		r, n, err := l.peekRune()
		if err != nil {
			return err
		}
		l.skip(n)
		if isNewline(r) {
			return nil
		}
	}
}

func (l *Lexer) skipMultiLine() error {
	for {
		b, err := l.readByte()
		if err != nil {
			return err
		}
		for b == '*' {
			b, err = l.readByte()
			if err != nil {
				return err
			}
			if b == '/' {
				return nil
			}
		}
	}
}

func (l *Lexer) scanString() (string, error) {
	var s strings.Builder
	for {
		c, err := l.peek()
		if err != nil {
			return "", err
		}
		if c == '"' {
			return s.String(), nil
		}
		l.skip(1)
		if c == '\n' {
			return "", errors.New("unescaped linebreak in string literal")
		}
		if c == '\\' {
			c, err = l.readByte()
			if err != nil {
				if err == io.EOF {
					err = errors.New("unterminated string")
				}
				return "", err
			}
			//XXX what about \u{}
			switch c {
			case '"', '\\': // nothing

			case 'b':
				c = '\b'
			case 'f':
				c = '\f'
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			default:
				s.WriteByte('\\')
			}
		}
		s.WriteByte(c)
	}
}

var newline = []byte{'\n'}

func (l *Lexer) scanBacktickString(keepIndentation bool) (string, error) {
	buf, err := l.skipTo('`')
	if err != nil {
		return "", err
	}
	n := len(buf) - 1
	out := buf[:n]
	if !keepIndentation {
		out = l.indentation.ReplaceAll(out, newline)
		if out[0] == '\n' {
			out = out[1:]
		}
	}
	l.skip(n)
	return string(out), nil
}

func (l *Lexer) scanTypeName() (string, error) {
	var s strings.Builder
	for {
		r, n, err := l.peekRune()
		if err != nil {
			return "", err
		}
		if !zng.TypeChar(r) {
			return s.String(), nil
		}
		s.WriteRune(r)
		l.skip(n)
	}
}

func (l *Lexer) scanIdentifier() (string, error) {
	s, err := l.scanTypeName()
	if err != nil {
		return "", err
	}
	if !zng.IsIdentifier(s) {
		s = ""
	}
	return s, nil
}

func (l *Lexer) peekPrimitive() (string, error) {
	return string(l.primitive.Find(l.cursor)), nil
}
