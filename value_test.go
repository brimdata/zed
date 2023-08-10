package zed_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/stretchr/testify/assert"
)

func BenchmarkValueUnder(b *testing.B) {
	var tmpVal zed.Value
	b.Run("primitive", func(b *testing.B) {
		val := zed.Null
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val.Under(&tmpVal)
		}
	})
	b.Run("named", func(b *testing.B) {
		typ, _ := zed.NewContext().LookupTypeNamed("name", zed.TypeNull)
		val := zed.NewValue(typ, nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val.Under(&tmpVal)
		}
	})
}

func TestValueValidate(t *testing.T) {
	recType := zed.NewTypeRecord(0, []zed.Field{
		zed.NewField("f", zed.NewTypeSet(0, zed.TypeString)),
	})
	t.Run("set/error/duplicate-element", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.Append([]byte("dup"))
		b.Append([]byte("dup"))
		// Don't normalize.
		b.EndContainer()
		val := zed.NewValue(recType, b.Bytes())
		assert.EqualError(t, val.Validate(), "invalid ZNG: duplicate set element")
	})
	t.Run("set/error/unsorted-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.Append([]byte("a"))
		b.Append([]byte("z"))
		b.Append([]byte("b"))
		// Don't normalize.
		b.EndContainer()
		val := zed.NewValue(recType, b.Bytes())
		assert.EqualError(t, val.Validate(), "invalid ZNG: set elements not sorted")
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
		val := zed.NewValue(recType, b.Bytes())
		assert.NoError(t, val.Validate())
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
			zed.NewTypeRecord(0, []zed.Field{
				zed.NewField("f", zed.NewTypeSet(0, zed.NewTypeRecord(0, []zed.Field{
					zed.NewField("g", zed.TypeString),
				}))),
			}),
			b.Bytes())
		assert.NoError(t, r.Validate())
	})
}
