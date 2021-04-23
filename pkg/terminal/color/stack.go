package color

import (
	"bytes"
	"strings"
)

type Stack []Code

func (s *Stack) Push(c Code) string {
	if !Enabled {
		return ""
	}
	if *s == nil {
		*s = make(Stack, 0, 4)
	}
	*s = append(*s, c)
	return c.String()
}

func (s *Stack) Pop() string {
	n := len(*s)
	if n == 1 {
		*s = (*s)[:0]
		return Reset.String()
	}
	tos := (*s)[n-1]
	next := (*s)[n-2]
	*s = (*s)[0 : n-1]
	if tos == next {
		return ""
	}
	return next.String()
}

func (s Stack) Top() Code {
	return s[len(s)-1]
}

func (s *Stack) ColorStart(b *strings.Builder, code Code) {
	if !Enabled {
		return
	}
	s.Push(code)
	b.WriteString(code.String())
}

func (s *Stack) ColorEnd(b *strings.Builder) {
	if !Enabled {
		return
	}
	op := s.Pop()
	if op != "" {
		b.WriteString(op)
	}
}

func (s *Stack) StartInBytes(b *bytes.Buffer, code Code) {
	if !Enabled {
		return
	}
	s.Push(code)
	b.WriteString(code.String())
}

func (s *Stack) EndInBytes(b *bytes.Buffer) {
	if !Enabled {
		return
	}
	op := s.Pop()
	if op != "" {
		b.WriteString(op)
	}
}

func (s *Stack) StartInString(b *strings.Builder, code Code) {
	if !Enabled {
		return
	}
	s.Push(code)
	b.WriteString(code.String())
}

func (s *Stack) EndInString(b *strings.Builder) {
	if !Enabled {
		return
	}
	op := s.Pop()
	if op != "" {
		b.WriteString(op)
	}
}
