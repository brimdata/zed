package zng_test

import (
	"testing"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/stretchr/testify/assert"
)

func TestRecordTypeCheck(t *testing.T) {
	r := zng.Record{
		Type: zng.NewTypeRecord(0, []zng.Column{
			zng.NewColumn("f", zng.NewTypeSet(0, zng.TypeString)),
		}),
	}
	t.Run("set/error/container-element", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.AppendContainer(nil)
		b.TransformContainer(zng.NormalizeSet)
		b.EndContainer()
		r.Raw = b.Bytes()
		assert.EqualError(t, r.TypeCheck(), "<set element> (set[string]): expected primitive type, got container")
	})
	t.Run("set/error/duplicate-element", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("dup"))
		// Don't normalize.
		b.EndContainer()
		r.Raw = b.Bytes()
		assert.EqualError(t, r.TypeCheck(), "<set element> (set[string]): duplicate element")
	})
	t.Run("set/error/unsorted-elements", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.AppendPrimitive([]byte("a"))
		b.AppendPrimitive([]byte("z"))
		b.AppendPrimitive([]byte("b"))
		// Don't normalize.
		b.EndContainer()
		r.Raw = b.Bytes()
		assert.EqualError(t, r.TypeCheck(), "<set element> (set[string]): elements not sorted")
	})
	t.Run("set/no-error", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("z"))
		b.AppendPrimitive([]byte("a"))
		b.TransformContainer(zng.NormalizeSet)
		b.EndContainer()
		r.Raw = b.Bytes()
		assert.NoError(t, r.TypeCheck())
	})
}
