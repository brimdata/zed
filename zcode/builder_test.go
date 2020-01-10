package zcode

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	empty := []byte{}
	unset := []byte(nil)
	t.Run("AppendUnsetContainer", func(t *testing.T) {
		b := NewBuilder()
		b.AppendUnsetContainer()
		expected := AppendContainerValue(nil, unset)
		require.Exactly(t, expected, b.Encode())
	})
	t.Run("AppendUnsetValue", func(t *testing.T) {
		b := NewBuilder()
		b.AppendUnsetValue()
		expected := AppendValue(nil, unset)
		require.Exactly(t, expected, b.Encode())
	})
	t.Run("Append", func(t *testing.T) {
		b := NewBuilder()
		b.Append(unset, true)
		b.Append(unset, false)
		expected := Append(nil, unset, true)
		expected = Append(expected, unset, false)
		require.Exactly(t, expected, b.Encode())
		b.Append(empty, true)
		b.Append(empty, false)
		expected = Append(expected, empty, true)
		expected = Append(expected, empty, false)
		require.Exactly(t, expected, b.Encode())
		v1, v2 := []byte("1"), []byte("22")
		b.Append(v1, true)
		b.Append(v2, false)
		expected = Append(expected, v1, true)
		expected = Append(expected, v2, false)
		require.Exactly(t, expected, b.Encode())
	})
	t.Run("BeginContainer/empty", func(t *testing.T) {
		b := NewBuilder()
		b.BeginContainer()
		b.EndContainer()
		expected := AppendContainerValue(nil, empty)
		require.Exactly(t, expected, b.Encode())
	})
	t.Run("BeginContainer/nonempty", func(t *testing.T) {
		b := NewBuilder()
		big := bytes.Repeat([]byte("x"), 256)
		v1, v2, v3, v4 := []byte("1"), []byte("22"), []byte("333"), []byte("4444")
		b.Append([]byte(v1), false)
		b.BeginContainer() // Start of outer container.
		b.Append([]byte(v2), false)
		b.BeginContainer()
		b.Append([]byte(big), false)
		b.EndContainer()
		b.Append([]byte(v3), false)
		b.EndContainer() // End of outer container.
		b.Append([]byte(v4), false)
		outerContainerValue := AppendValue(nil, v2)
		outerContainerValue = AppendContainer(outerContainerValue, [][]byte{big})
		outerContainerValue = AppendValue(outerContainerValue, v3)
		expected := AppendValue(nil, v1)
		expected = AppendContainerValue(expected, outerContainerValue)
		expected = AppendValue(expected, v4)
		require.Exactly(t, expected, b.Encode())
	})
	t.Run("BeginContainer/panic", func(t *testing.T) {
		b := NewBuilder()
		b.BeginContainer()
		require.Panics(t, func() { b.Encode() })
	})
	t.Run("EndContainer", func(t *testing.T) {
		b := NewBuilder()
		require.Panics(t, func() { b.EndContainer() })
	})
	t.Run("Reset", func(t *testing.T) {
		b := NewBuilder()
		b.Append([]byte("1"), false)
		b.Reset()
		require.Nil(t, b.Encode())
	})
}
