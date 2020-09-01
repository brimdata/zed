package zngio

import (
	"io"
	"sync"

	"github.com/brimsec/zq/zio"
)

type buffer struct {
	data []byte
	off  int
}

var bufferPool sync.Pool

func newBuffer(length int) *buffer {
	if length <= zio.DefaultZngLZ4BlockSize {
		b, ok := bufferPool.Get().(*buffer)
		if !ok {
			b = &buffer{data: make([]byte, zio.DefaultZngLZ4BlockSize)}
		}
		b.data = b.data[:length]
		b.off = 0
		return b
	}
	return &buffer{data: make([]byte, length)}
}

func (b *buffer) free() {
	if cap(b.data) == zio.DefaultZngLZ4BlockSize {
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

func (b *buffer) next(n int) []byte {
	if l := b.length(); n > l {
		n = l
	}
	off := b.off
	b.off += n
	return b.data[off:b.off]
}

func (b *buffer) ReadByte() (byte, error) {
	if b.length() < 1 {
		return 0, io.EOF
	}
	off := b.off
	b.off++
	return b.data[off], nil
}
