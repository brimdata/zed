package storage

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeeker(t *testing.T) {
	r := NewBytesReader([]byte{0, 1})
	s, err := NewSeeker(r)
	require.NoError(t, err)

	// Test that ReadAt doesn't affect the offset.
	b := make([]byte, 3)
	n, err := s.ReadAt(b, 1)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, 1, n)
	assert.EqualValues(t, 1, b[0])
	n64, err := s.Seek(0, io.SeekCurrent)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, n64)

	// Test Read followed by Seek to the beginning.
	for i := 0; i < 3; i++ {
		n, err = s.Read(b)
		assert.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.EqualValues(t, 0, b[0])
		assert.EqualValues(t, 1, b[1])
		n64, err = s.Seek(0, io.SeekStart)
		assert.NoError(t, err)
		assert.EqualValues(t, 0, n64)
	}
}
