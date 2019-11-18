package peeker

import (
	"errors"
	"io"
)

type Reader struct {
	io.Reader
	limit  int
	buffer []byte
	cursor []byte
	eof    bool
}

var (
	ErrBufferOverflow = errors.New("buffer too big")
	ErrTruncated      = errors.New("truncated input")
)

func NewReader(reader io.Reader, size, max int) *Reader {
	b := make([]byte, size)
	return &Reader{
		Reader: reader,
		limit:  max,
		buffer: b,
		cursor: b[:0],
	}
}

func (r *Reader) fill(min int) error {
	if min > r.limit {
		return ErrBufferOverflow
	}
	if min > cap(r.buffer) {
		r.buffer = make([]byte, min)
	}
	r.buffer = r.buffer[:cap(r.buffer)]
	copy(r.buffer, r.cursor)
	clen := len(r.cursor)
	space := len(r.buffer) - clen
	for space > 0 {
		cc, err := r.Reader.Read(r.buffer[clen:])
		if cc > 0 {
			clen += cc
			space -= cc
		}
		if err != nil {
			if err == io.EOF {
				r.eof = true
				break
			}
			return err
		}
	}
	r.buffer = r.buffer[:clen]
	r.cursor = r.buffer
	return nil
}

func (r *Reader) Peek(n int) ([]byte, error) {
	if len(r.cursor) == 0 && r.eof {
		return nil, io.EOF
	}
	if n > len(r.cursor) && !r.eof {
		if err := r.fill(n); err != nil {
			return nil, err
		}
	}
	if n > len(r.cursor) {
		return r.cursor, ErrTruncated
	}
	return r.cursor[:n], nil
}

func (r *Reader) Read(n int) ([]byte, error) {
	b, err := r.Peek(n)
	if err != nil {
		return nil, err
	}
	r.cursor = r.cursor[n:]
	return b, nil
}
