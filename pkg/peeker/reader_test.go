package peeker

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEOFOnEmptyFill(t *testing.T) {
	p := NewReader(strings.NewReader("0123456789"), 1024, 1024, false)
	n, err := p.Read(10)
	assert.NoError(t, err)
	assert.Len(t, n, 10)
	n, err = p.Peek(10)
	assert.ErrorIs(t, err, io.EOF)
	assert.Len(t, n, 0)
}

type drip struct {
	b []byte
}

func (d *drip) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, io.ErrShortBuffer
	}
	if len(d.b) == 0 {
		return 0, io.EOF
	}
	b[0] = d.b[0]
	d.b = d.b[1:]
	return 1, nil
}

func TestInteractive(t *testing.T) {
	p := NewReader(&drip{[]byte("0123456789")}, 1024, 1024, true)
	b, err := p.Peek(3)
	assert.NoError(t, err)
	assert.Equal(t, "012", string(b))
	assert.Equal(t, 3, len(p.buffer))
}

func TestNonInteractive(t *testing.T) {
	p := NewReader(&drip{[]byte("0123456789")}, 1024, 1024, false)
	b, err := p.Peek(3)
	assert.NoError(t, err)
	assert.Equal(t, "012", string(b))
	assert.Equal(t, 10, len(p.buffer))
}
