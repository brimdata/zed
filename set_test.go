package zed_test

import (
	"bytes"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSet(t *testing.T) {
	t.Run("duplicate-element", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.Append([]byte("dup"))
		b.Append([]byte("dup"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, []byte("dup"))
		expected := zcode.Append(nil, set)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("unsorted-elements", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.Append([]byte("z"))
		b.Append([]byte("a"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, []byte("a"))
		set = zcode.Append(set, []byte("z"))
		expected := zcode.Append(nil, set)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("unsorted-and-duplicate-elements", func(t *testing.T) {
		b := zcode.NewBuilder()
		big := bytes.Repeat([]byte("x"), 256)
		small := []byte("small")
		b.Append(big)
		b.BeginContainer()
		// Append duplicate elements in reverse of set-normal order.
		for i := 0; i < 3; i++ {
			b.Append(big)
			b.Append(big)
			b.Append(small)
			b.Append(small)
			b.Append(nil)
			b.Append(nil)
		}
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, nil)
		set = zcode.Append(set, small)
		set = zcode.Append(set, big)
		expected := zcode.Append(nil, big)
		expected = zcode.Append(expected, set)
		require.Exactly(t, expected, b.Bytes())
	})
}
