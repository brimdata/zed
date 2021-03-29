package zng_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zcode"
	"github.com/brimdata/zq/zio/tzngio"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordTypeCheck(t *testing.T) {
	r := zng.NewRecord(
		zng.NewTypeRecord(0, []zng.Column{
			zng.NewColumn("f", zng.NewTypeSet(0, zng.TypeString)),
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
		assert.EqualError(t, r.TypeCheck(), "<set element> (set[string]): duplicate element")
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
		assert.EqualError(t, r.TypeCheck(), "<set element> (set[string]): elements not sorted")
	})
	t.Run("set/primitive-elements", func(t *testing.T) {
		var b zcode.Builder
		b.BeginContainer()
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("dup"))
		b.AppendPrimitive([]byte("z"))
		b.AppendPrimitive([]byte("a"))
		b.TransformContainer(zng.NormalizeSet)
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
		b.TransformContainer(zng.NormalizeSet)
		b.EndContainer()
		r := zng.NewRecord(
			zng.NewTypeRecord(0, []zng.Column{
				zng.NewColumn("f", zng.NewTypeSet(0, zng.NewTypeRecord(0, []zng.Column{
					zng.NewColumn("g", zng.TypeString),
				}))),
			}),
			b.Bytes())
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

func TestRecordTs(t *testing.T) {
	cases := []struct {
		typ, val string
		expected nano.Ts
	}{
		{"record[ts:time]", "[1;]", nano.Ts(time.Second)},
		{"record[notts:time]", "[1;]", nano.MinTs}, // No ts field.
		{"record[ts:time]", "[-;]", nano.MinTs},    // Null ts field.
		{"record[ts:int64]", "[1;]", nano.MinTs},   // Type of ts field is not TypeOfTime.
	}
	for _, c := range cases {
		input := fmt.Sprintf("#0:%s\n0:%s\n", c.typ, c.val)
		zr := tzngio.NewReader(strings.NewReader(input), resolver.NewContext())
		rec, err := zr.Read()
		assert.NoError(t, err)
		require.NotNil(t, rec)
		assert.Exactly(t, c.expected, rec.Ts(), "input: %q", input)
	}
}
