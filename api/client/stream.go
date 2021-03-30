package client

import (
	"bufio"
	"errors"
	"io"

	"github.com/brimdata/zed/api"
)

type Stream struct {
	scanner *bufio.Scanner
}

type Payloads []interface{}

func (p Payloads) Error() error {
	last := p[len(p)-1]
	if te, ok := last.(*api.TaskEnd); ok {
		if te.Error != nil {
			return te.Error
		}
		return nil
	}
	return errors.New("expected last payload to be of type *TaskEnd")
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

func (s *Stream) ReadAll() (Payloads, error) {
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
		if _, ok := v.(*api.TaskEnd); ok {
			return payloads, nil
		}
	}
}
