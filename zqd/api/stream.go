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

func ReadStream(s *Stream) ([]interface{}, error) {
	var payloads []interface{}
	for {
		v, err := s.Next()
		if err != nil {
			if err == io.EOF {
				return payloads, nil
			}
			return nil, err
		}
		payloads = append(payloads, v)
		switch payload := v.(type) {
		case *TaskEnd:
			if payload.Error != nil {
				return nil, payload.Error
			}
			return payloads, nil
		}
	}
}
