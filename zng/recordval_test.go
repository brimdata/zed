package zng_test

import (
	"strings"
	"testing"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

const in = `
#zfile=string
#zbool=bool
#0:record[foo:zfile,bar:zbool]
0:[hello;true;]
`

func TestRecordAccessAlias(t *testing.T) {
	reader := tzngio.NewReader(strings.NewReader(in), resolver.NewContext())
	rec, err := reader.Read()
	require.NoError(t, err)
	s, err := rec.AccessString("foo")
	require.NoError(t, err)
	assert.Equal(t, s, "hello")
	b, err := rec.AccessBool("bar")
	require.NoError(t, err)
	assert.Equal(t, b, true)
}
