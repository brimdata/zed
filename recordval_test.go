package zed_test

import (
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordTypeCheck(t *testing.T) {
	r := zed.NewValue(
		zed.NewTypeRecord(0, []zed.Column{
			zed.NewColumn("f", zed.NewTypeSet(0, zed.TypeString)),
		}),
		nil)
	t.Run("set/error/duplicate-element", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("dup"))
		// Don't normalize.
		b.EndContainer()
		r.Bytes = b.Bytes()
		assert.EqualError(t, r.TypeCheck(), "<set element> (|[string]|): duplicate element")
	})
	t.Run("set/error/unsorted-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.AppendPrimitive([]byte("a"))
		b.AppendPrimitive([]byte("z"))
		b.AppendPrimitive([]byte("b"))
		// Don't normalize.
		b.EndContainer()
		r.Bytes = b.Bytes()
		assert.EqualError(t, r.TypeCheck(), "<set element> (|[string]|): elements not sorted")
	})
	t.Run("set/primitive-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("z"))
		b.AppendPrimitive([]byte("a"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		r.Bytes = b.Bytes()
		assert.NoError(t, r.TypeCheck())
	})
	t.Run("set/complex-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		for _, s := range []string{"dup", "dup", "z", "a"} {
			b.BeginContainer()
			b.AppendPrimitive([]byte(s))
			b.EndContainer()
		}
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		r := zed.NewValue(
			zed.NewTypeRecord(0, []zed.Column{
				zed.NewColumn("f", zed.NewTypeSet(0, zed.NewTypeRecord(0, []zed.Column{
					zed.NewColumn("g", zed.TypeString),
				}))),
			}),
			b.Bytes())
		assert.NoError(t, r.TypeCheck())
	})

}

func TestRecordAccessAlias(t *testing.T) {
	const input = `{foo:"hello" (=zfile),bar:true (=zbool)} (=0)`
	reader := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	rec, err := reader.Read()
	require.NoError(t, err)
	s, err := rec.AccessString("foo")
	require.NoError(t, err)
	assert.Equal(t, s, "hello")
	b, err := rec.AccessBool("bar")
	require.NoError(t, err)
	assert.Equal(t, b, true)
}
