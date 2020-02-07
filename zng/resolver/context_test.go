package resolver

import (
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextAddColumns(t *testing.T) {
	ctx := NewContext()
	d := ctx.LookupTypeRecord([]zng.Column{zng.NewColumn("s1", zng.TypeString)})
	r, err := zbuf.NewRecordZeekStrings(d, "S1")
	require.NoError(t, err)
	cols := []zng.Column{zng.NewColumn("ts", zng.TypeTime), zng.NewColumn("s2", zng.TypeString)}
	ts, _ := nano.Parse([]byte("123.456"))
	r, err = ctx.AddColumns(r, cols, []zng.Value{zng.NewTime(ts), zng.NewString("S2")})
	require.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "S1", r.Value(0).String())
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "S2", r.Value(2).String())
	zv, _ := r.Slice(4)
	assert.Nil(t, zv)
}

func TestDuplicates(t *testing.T) {
	ctx := NewContext()
	setType := ctx.LookupTypeSet(zng.TypeInt)
	typ1 := ctx.LookupTypeRecord([]zng.Column{
		zng.NewColumn("a", zng.TypeString),
		zng.NewColumn("b", setType),
	})
	typ2, err := ctx.LookupByName("record[a:string,b:set[int]]")
	require.NoError(t, err)
	assert.EqualValues(t, typ1.ID(), typ2.ID())
	assert.EqualValues(t, setType.ID(), typ2.(*zng.TypeRecord).Columns[1].Type.ID())
	typ3, err := ctx.LookupByName("set[int]")
	require.NoError(t, err)
	assert.Equal(t, setType.ID(), typ3.ID())
}
