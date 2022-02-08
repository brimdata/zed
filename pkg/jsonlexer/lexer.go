package jsonlexer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

type Token byte

const (
	TokenErr Token = iota

	TokenBeginArray
	TokenBeginObject
	TokenEndArray
	TokenEndObject
	TokenNameSeparator
	TokenValueSeparator

	TokenString
	TokenNumber
	TokenTrue
	TokenFalse
	TokenNull
)

type Lexer struct {
	br  *bufio.Reader
	buf []byte
	err error
}

func New(br *bufio.Reader) *Lexer {
	return &Lexer{
		br:  br,
		buf: make([]byte, 0, 128),
	}
}

func (l *Lexer) Buf() []byte {
	return l.buf
}

func (l *Lexer) Err() error {
	return l.err
}

func (l *Lexer) Token() Token {
	l.buf = l.buf[:1]
	for {
		c, err := l.br.ReadByte()
		if err != nil {
			l.err = err
			return TokenErr
		}
		l.buf[0] = c
		// Cases are ordered by decreasing expected frequency.
		switch c {
		case '"':
			return l.readString()
		case ',':
			return TokenValueSeparator
		case ':':
			return TokenNameSeparator
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			return l.readNumber(c)
		case 'f':
			return l.readLiteral("alse", TokenFalse)
		case 't':
			return l.readLiteral("rue", TokenTrue)
		case 'n':
			return l.readLiteral("ull", TokenNull)
		case '{':
			return TokenBeginObject
		case '}':
			return TokenEndObject
		case '[':
			return TokenBeginArray
		case ']':
			return TokenEndArray
		case ' ', '\n', '\r', '\t':
			continue
		default:
			l.err = fmt.Errorf("invalid character %q looking for beginning of value", c)
			return TokenErr
		}
	}
}

func (l *Lexer) readLiteral(s string, t Token) Token {
	for i := range s {
		c, err := l.br.ReadByte()
		if err != nil {
			l.err = err
			return TokenErr
		}
		if s[i] != c {
			l.err = errors.New("bad literal name")
			return TokenErr
		}
	}
	c, err := l.br.ReadByte()
	if err != nil {
		if err == io.EOF {
			return t
		}
	}
	if err := l.br.UnreadByte(); err != nil {
		l.err = err
		return TokenErr
	}
	if !isValueBoundaryChar(c) {
		l.err = errors.New("bad literal name")
		return TokenErr
	}
	return t
}

func isValueBoundaryChar(c byte) bool {
	switch c {
	case '[', '{', ']', '}', ',', '"':
		return true
	case 0x20, 0x0a, 0x0d, 0x09:
		return true
	}
	return false
}

func (l *Lexer) readNumber(c byte) Token {
	l.buf = append(l.buf[:0], c)
	for {
		c, err := l.br.ReadByte()
		if err != nil {
			if err == io.EOF {
				return TokenNumber
			}
			l.err = err
			return TokenErr
		}
		if !isNumberChar(c) {
			if err := l.br.UnreadByte(); err != nil {
				l.err = err
				return TokenErr
			}
			return TokenNumber
		}
		l.buf = append(l.buf, c)
	}
}

func isNumberChar(c byte) bool {
	return c-'0' < 10 || c == '.' || c == 'e' || c == 'E' || c == '-' || c == '+'
}

func (l *Lexer) readString() Token {
	var escape bool
	for {
		c, err := l.br.ReadByte()
		if err != nil {
			l.err = err
			return TokenErr
		}
		l.buf = append(l.buf, c)
		if c == '"' && !escape {
			return TokenString
		}
		escape = c == '\\' && !escape
	}
}
