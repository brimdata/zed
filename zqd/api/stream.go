package api

import (
	"bufio"
	"io"
)

type Stream struct {
	scanner *bufio.Scanner
}

func NewStream(s *bufio.Scanner) *Stream {
	return &Stream{s}
}

func (s *Stream) Next() (interface{}, error) {
	if s.scanner.Scan() {
		v, err := unpack(s.scanner.Bytes())
		if err != nil {
			return nil, err
		}
		return v, err
	}
	err := s.scanner.Err()
	if err != io.EOF {
		return nil, err
	}
	return nil, nil
}
