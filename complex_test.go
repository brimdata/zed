package zed_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordAccessNamed(t *testing.T) {
	const input = `{foo:"hello" (=zfile),bar:true (=zbool)} (=0)`
	reader := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	rec, err := reader.Read()
	require.NoError(t, err)
	s := rec.Deref("foo").AsString()
	assert.Equal(t, s, "hello")
	b := rec.Deref("bar").AsBool()
	assert.Equal(t, b, true)
}

func TestNonRecordDeref(t *testing.T) {
	const input = `
1
192.168.1.1
null
[1,2,3]
|[1,2,3]|`
	reader := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	for {
		val, err := reader.Read()
		if val == nil {
			break
		}
		require.NoError(t, err)
		v := val.Deref("foo")
		require.Nil(t, v)
	}
}

func TestNormalizeSet(t *testing.T) {
	t.Run("duplicate-element", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.Append([]byte("dup"))
		b.Append([]byte("dup"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, []byte("dup"))
		expected := zcode.Append(nil, set)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("unsorted-elements", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.Append([]byte("z"))
		b.Append([]byte("a"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, []byte("a"))
		set = zcode.Append(set, []byte("z"))
		expected := zcode.Append(nil, set)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("unsorted-and-duplicate-elements", func(t *testing.T) {
		b := zcode.NewBuilder()
		big := bytes.Repeat([]byte("x"), 256)
		small := []byte("small")
		b.Append(big)
		b.BeginContainer()
		// Append duplicate elements in reverse of set-normal order.
		for i := 0; i < 3; i++ {
			b.Append(big)
			b.Append(big)
			b.Append(small)
			b.Append(small)
			b.Append(nil)
			b.Append(nil)
		}
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, nil)
		set = zcode.Append(set, small)
		set = zcode.Append(set, big)
		expected := zcode.Append(nil, big)
		expected = zcode.Append(expected, set)
		require.Exactly(t, expected, b.Bytes())
	})
}

func TestDuplicates(t *testing.T) {
	ctx := zed.NewContext()
	setType := ctx.LookupTypeSet(zed.TypeInt32)
	typ1, err := ctx.LookupTypeRecord([]zed.Column{
		zed.NewColumn("a", zed.TypeString),
		zed.NewColumn("b", setType),
	})
	require.NoError(t, err)
	typ2, err := zson.ParseType(ctx, "{a:string,b:|[int32]|}")
	require.NoError(t, err)
	assert.EqualValues(t, typ1.ID(), typ2.ID())
	assert.EqualValues(t, setType.ID(), typ2.(*zed.TypeRecord).Columns[1].Type.ID())
	typ3, err := ctx.LookupByValue(zed.EncodeTypeValue(setType))
	require.NoError(t, err)
	assert.Equal(t, setType.ID(), typ3.ID())
}

func TestTranslateNamed(t *testing.T) {
	c1 := zed.NewContext()
	c2 := zed.NewContext()
	set1, err := zson.ParseType(c1, "|[int64]|")
	require.NoError(t, err)
	set2, err := zson.ParseType(c2, "|[int64]|")
	require.NoError(t, err)
	named1, err := c1.LookupTypeNamed("foo", set1)
	require.NoError(t, err)
	named2, err := c2.LookupTypeNamed("foo", set2)
	require.NoError(t, err)
	named3, err := c2.TranslateType(named1)
	require.NoError(t, err)
	assert.Equal(t, named2, named3)
}

func TestCopyMutateColumns(t *testing.T) {
	c := zed.NewContext()
	cols := []zed.Column{{"foo", zed.TypeString}, {"bar", zed.TypeInt64}}
	typ, err := c.LookupTypeRecord(cols)
	require.NoError(t, err)
	cols[0].Type = nil
	require.NotNil(t, typ.Columns[0].Type)
}
