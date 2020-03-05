package api

import (
	"bufio"
	"bytes"
	"io"
)

var sep = []byte("\n\n")

func NewJSONPipeScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Split(splitJSONPipe)
	return s
}

func splitJSONPipe(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, sep); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
