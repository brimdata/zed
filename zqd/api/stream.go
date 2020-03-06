package api

import (
	"bufio"
	"context"
	"io"
)

type Stream struct {
	scanner *bufio.Scanner
	cancel  context.CancelFunc
}

func NewStream(s *bufio.Scanner, c context.CancelFunc) *Stream {
	return &Stream{s, c}
}

func (s *Stream) end() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *Stream) Next() (interface{}, error) {
	if s.scanner.Scan() {
		v, err := unpack(s.scanner.Bytes())
		if err != nil {
			s.end()
			return nil, err
		}
		return v, err
	}
	s.end()
	err := s.scanner.Err()
	if err != io.EOF {
		return nil, err
	}
	return nil, nil
}
