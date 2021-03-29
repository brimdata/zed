package slicer_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/brimdata/zq/pkg/slicer"
	"github.com/stretchr/testify/assert"
)

func TestSlicer(t *testing.T) {
	in := []byte("abcdefghijklmnopqrstuvwxyz")
	slices := []slicer.Slice{
		{0, 2},
		{0, 26},
		{3, 4},
		{25, 1},
		{25, 2},
	}
	expected := []byte("ababcdefghijklmnopqrstuvwxyzdefgzz")
	reader, err := slicer.NewReader(bytes.NewReader(in), slices)
	assert.NoError(t, err)
	out, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)
	assert.Exactly(t, expected, out)
}
