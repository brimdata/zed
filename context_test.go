package zed_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
