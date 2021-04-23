package color

import (
	"io"
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

func (s *Stack) Start(w io.Writer, code Code) error {
	if !Enabled {
		return nil
	}
	s.Push(code)
	_, err := io.WriteString(w, code.String())
	return err
}

func (s *Stack) End(w io.Writer) error {
	if !Enabled {
		return nil
	}
	var err error
	if op := s.Pop(); op != "" {
		_, err = io.WriteString(w, op)
	}
	return err
}
