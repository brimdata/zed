package lineio

import (
	"bufio"
	"io"

	"github.com/brimdata/zed"
)

type Reader struct {
	scanner *bufio.Scanner
}

func NewReader(r io.Reader) *Reader {
	return &Reader{scanner: bufio.NewScanner(r)}
}

func (r *Reader) Read() (*zed.Value, error) {
	if !r.scanner.Scan() || r.scanner.Err() != nil {
		return nil, r.scanner.Err()
	}
	return zed.NewValue(zed.TypeString, r.scanner.Bytes()), nil
}
