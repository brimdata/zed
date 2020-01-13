package zcode

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	empty := []byte{}
	unset := []byte(nil)
	v1, v2 := []byte("1"), []byte("22")
	t.Run("AppendContainer", func(t *testing.T) {
		b := NewBuilder()
		b.AppendContainer(empty)
		expected := AppendContainer(nil, empty)
		require.Exactly(t, expected, b.Bytes())
		b.AppendContainer(unset)
		expected = AppendContainer(expected, unset)
		require.Exactly(t, expected, b.Bytes())
		b.AppendContainer(v1)
		b.AppendContainer(v2)
		expected = AppendContainer(expected, v1)
		expected = AppendContainer(expected, v2)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("AppendSimple", func(t *testing.T) {
		b := NewBuilder()
		b.AppendSimple(empty)
		expected := AppendSimple(nil, empty)
		require.Exactly(t, expected, b.Bytes())
		b.AppendSimple(unset)
		expected = AppendSimple(expected, unset)
		require.Exactly(t, expected, b.Bytes())
		b.AppendSimple(v1)
		b.AppendSimple(v2)
		expected = AppendSimple(expected, v1)
		expected = AppendSimple(expected, v2)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("BeginContainer/empty", func(t *testing.T) {
		b := NewBuilder()
		b.BeginContainer()
		b.EndContainer()
		expected := AppendContainer(nil, empty)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("BeginContainer/nonempty", func(t *testing.T) {
		b := NewBuilder()
		big := bytes.Repeat([]byte("x"), 256)
		v1, v2, v3, v4 := []byte("1"), []byte("22"), []byte("333"), []byte("4444")
		b.AppendSimple(v1)
		b.BeginContainer() // Start of outer container.
		b.AppendSimple(v2)
		b.BeginContainer() // Start of inner container.
		b.AppendSimple(big)
		b.EndContainer() // End of inner container.
		b.AppendSimple(v3)
		b.EndContainer() // End of outer container.
		b.AppendSimple(v4)
		innerContainer := AppendSimple(nil, big)
		outerContainer := AppendSimple(nil, v2)
		outerContainer = AppendContainer(outerContainer, innerContainer)
		outerContainer = AppendSimple(outerContainer, v3)
		expected := AppendSimple(nil, v1)
		expected = AppendContainer(expected, outerContainer)
		expected = AppendSimple(expected, v4)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("BeginContainer/panic", func(t *testing.T) {
		b := NewBuilder()
		b.BeginContainer()
		require.Panics(t, func() { b.Bytes() })
	})
	t.Run("EndContainer", func(t *testing.T) {
		b := NewBuilder()
		require.Panics(t, func() { b.EndContainer() })
	})
	t.Run("Reset", func(t *testing.T) {
		b := NewBuilder()
		b.AppendSimple([]byte("1"))
		b.Reset()
		require.Nil(t, b.Bytes())
	})
}
