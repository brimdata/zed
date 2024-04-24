package lineio

import (
	"bufio"
	"io"

	"github.com/brimdata/zed"
)

type Reader struct {
	scanner *bufio.Scanner
	arena   *zed.Arena
	val     zed.Value
}

func NewReader(r io.Reader) *Reader {
	s := bufio.NewScanner(r)
	s.Buffer(nil, 25*1024*1024)
	return &Reader{
		scanner: bufio.NewScanner(r),
		arena:   zed.NewArena(),
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	if !r.scanner.Scan() || r.scanner.Err() != nil {
		return nil, r.scanner.Err()
	}
	r.arena.Reset()
	r.val = r.arena.NewString(r.scanner.Text())
	return &r.val, nil
}
