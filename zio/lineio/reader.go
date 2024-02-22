package lineio

import (
	"bufio"
	"io"

	"github.com/brimdata/zed"
)

type Reader struct {
	scanner *bufio.Scanner
	val     zed.Value
}

func NewReader(r io.Reader) *Reader {
	s := bufio.NewScanner(r)
	s.Buffer(nil, 25*1024*2014)
	return &Reader{scanner: s}
}

func (r *Reader) Read() (*zed.Value, error) {
	if !r.scanner.Scan() || r.scanner.Err() != nil {
		return nil, r.scanner.Err()
	}
	r.val = zed.NewString(r.scanner.Text())
	return &r.val, nil
}
