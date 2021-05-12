package zngio

import (
	"io"
	"sync"
)

type buffer struct {
	data []byte
	off  int
}

var bufferPool sync.Pool

func newBuffer(length int) *buffer {
	if length <= DefaultLZ4BlockSize {
		b, ok := bufferPool.Get().(*buffer)
		if !ok {
			b = &buffer{data: make([]byte, DefaultLZ4BlockSize)}
		}
		b.data = b.data[:length]
		b.off = 0
		return b
	}
	return &buffer{data: make([]byte, length)}
}

func (b *buffer) free() {
	if cap(b.data) == DefaultLZ4BlockSize {
		bufferPool.Put(b)
	}
}

// Bytes is so named to avoid collision with the bytes package.
func (b *buffer) Bytes() []byte {
	return b.data[b.off:]
}

func (b *buffer) length() int {
	return len(b.data) - b.off
}

func (b *buffer) read(n int) ([]byte, error) {
	var err error
	if avail := b.length(); n > avail {
		if avail == 0 {
			return nil, io.EOF
		}
		err = io.ErrUnexpectedEOF
		n = avail
	}
	off := b.off
	b.off += n
	return b.data[off:b.off], err
}

func (b *buffer) ReadByte() (byte, error) {
	if b.length() < 1 {
		return 0, io.EOF
	}
	off := b.off
	b.off++
	return b.data[off], nil
}
