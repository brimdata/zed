package zed_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/stretchr/testify/assert"
)

func TestValueValidate(t *testing.T) {
	r := zed.NewValue(
		zed.NewTypeRecord(0, []zed.Column{
			zed.NewColumn("f", zed.NewTypeSet(0, zed.TypeString)),
		}),
		nil)
	t.Run("set/error/duplicate-element", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.Append([]byte("dup"))
		b.Append([]byte("dup"))
		// Don't normalize.
		b.EndContainer()
		r.Bytes = b.Bytes()
		assert.EqualError(t, r.Validate(), "invalid ZNG: duplicate set element")
	})
	t.Run("set/error/unsorted-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.Append([]byte("a"))
		b.Append([]byte("z"))
		b.Append([]byte("b"))
		// Don't normalize.
		b.EndContainer()
		r.Bytes = b.Bytes()
		assert.EqualError(t, r.Validate(), "invalid ZNG: set elements not sorted")
	})
	t.Run("set/primitive-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.Append([]byte("dup"))
		b.Append([]byte("dup"))
		b.Append([]byte("z"))
		b.Append([]byte("a"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		r.Bytes = b.Bytes()
		assert.NoError(t, r.Validate())
	})
	t.Run("set/complex-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		for _, s := range []string{"dup", "dup", "z", "a"} {
			b.BeginContainer()
			b.Append([]byte(s))
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
		assert.NoError(t, r.Validate())
	})
}
