package zcode

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	empty := []byte{}
	null := []byte(nil)
	v1, v2 := []byte("1"), []byte("22")
	t.Run("AppendContainer", func(t *testing.T) {
		b := NewBuilder()
		b.AppendContainer(empty)
		expected := AppendContainer(nil, empty)
		require.Exactly(t, expected, b.Bytes())
		b.AppendContainer(null)
		expected = AppendContainer(expected, null)
		require.Exactly(t, expected, b.Bytes())
		b.AppendContainer(v1)
		b.AppendContainer(v2)
		expected = AppendContainer(expected, v1)
		expected = AppendContainer(expected, v2)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("AppendPrimitive", func(t *testing.T) {
		b := NewBuilder()
		b.AppendPrimitive(empty)
		expected := AppendPrimitive(nil, empty)
		require.Exactly(t, expected, b.Bytes())
		b.AppendPrimitive(null)
		expected = AppendPrimitive(expected, null)
		require.Exactly(t, expected, b.Bytes())
		b.AppendPrimitive(v1)
		b.AppendPrimitive(v2)
		expected = AppendPrimitive(expected, v1)
		expected = AppendPrimitive(expected, v2)
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
		b.AppendPrimitive(v1)
		b.BeginContainer() // Start of outer container.
		b.AppendPrimitive(v2)
		b.BeginContainer() // Start of inner container.
		b.AppendPrimitive(big)
		b.EndContainer() // End of inner container.
		b.AppendPrimitive(v3)
		b.EndContainer() // End of outer container.
		b.AppendPrimitive(v4)
		innerContainer := AppendPrimitive(nil, big)
		outerContainer := AppendPrimitive(nil, v2)
		outerContainer = AppendContainer(outerContainer, innerContainer)
		outerContainer = AppendPrimitive(outerContainer, v3)
		expected := AppendPrimitive(nil, v1)
		expected = AppendContainer(expected, outerContainer)
		expected = AppendPrimitive(expected, v4)
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
	t.Run("TransformContainer/empty", func(t *testing.T) {
		b := NewBuilder()
		b.BeginContainer()
		b.TransformContainer(func(body Bytes) Bytes { return AppendPrimitive(nil, v1) })
		b.EndContainer()
		expected := AppendContainer(nil, AppendPrimitive(nil, v1))
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("TransformContainer/nonempty", func(t *testing.T) {
		b := NewBuilder()
		b.BeginContainer()
		b.AppendPrimitive(v1)
		b.AppendPrimitive(v2)
		b.TransformContainer(func(body Bytes) Bytes { return AppendPrimitive(nil, v2) })
		b.EndContainer()
		expected := AppendContainer(nil, AppendPrimitive(nil, v2))
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("TransformContainer/panic", func(t *testing.T) {
		b := NewBuilder()
		require.Panics(t, func() {
			b.TransformContainer(func(body Bytes) Bytes { return nil })
		})
	})
	t.Run("Reset", func(t *testing.T) {
		b := NewBuilder()
		b.AppendPrimitive([]byte("1"))
		b.Reset()
		require.Nil(t, b.Bytes())
	})
}
