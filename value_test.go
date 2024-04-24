package zed_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStringNotNull(t *testing.T) {
	arena := zed.NewArena()
	defer arena.Unref()
	assert.NotNil(t, arena.NewString("").Bytes())
}

func BenchmarkValueUnder(b *testing.B) {
	b.Run("primitive", func(b *testing.B) {
		val := zed.Null
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val.Under(nil)
		}
	})
	b.Run("named", func(b *testing.B) {
		arena := zed.NewArena()
		defer arena.Unref()
		typ, _ := zed.NewContext().LookupTypeNamed("name", zed.TypeNull)
		val := arena.New(typ, nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			val.Under(arena)
		}
	})
}

func TestValueValidate(t *testing.T) {
	zctx := zed.NewContext()
	recType, err := zson.ParseType(zctx, "{f:|[string]|}")
	require.NoError(t, err)
	arena := zed.NewArena()
	defer arena.Unref()
	t.Run("set/error/duplicate-element", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.Append([]byte("dup"))
		b.Append([]byte("dup"))
		// Don't normalize.
		b.EndContainer()
		val := arena.New(recType, b.Bytes())
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
		val := arena.New(recType, b.Bytes())
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
		val := arena.New(recType, b.Bytes())
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
		typ, err := zson.ParseType(zctx, "{f:|[{g:string}]|}")
		require.NoError(t, err)
		val := arena.New(typ, b.Bytes())
		assert.NoError(t, val.Validate())
	})
}
