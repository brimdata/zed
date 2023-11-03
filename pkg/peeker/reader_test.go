package peeker

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEOFOnEmptyFill(t *testing.T) {
	p := NewReader(strings.NewReader("0123456789"), 1024, 1024)
	n, err := p.Read(10)
	assert.NoError(t, err)
	assert.Len(t, n, 10)
	n, err = p.Peek(10)
	assert.ErrorIs(t, err, io.EOF)
	assert.Len(t, n, 0)
}

func TestNegativeLength(t *testing.T) {
	p := NewReader(nil, 0, 0)
	_, err := p.Read(-1)
	assert.ErrorContains(t, err, "peeker: negative length")
	_, err = p.Peek(-1)
	assert.ErrorContains(t, err, "peeker: negative length")
}
