package resolver

import (
	"testing"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableAddColumns(t *testing.T) {
	ctx := NewContext()
	d := ctx.LookupByColumns([]zng.Column{zng.NewColumn("s1", zng.TypeString)})
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
