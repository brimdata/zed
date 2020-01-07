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
	tab := NewTable()
	d := tab.GetByColumns([]zng.Column{{"s1", zng.TypeString}})
	r, err := zbuf.NewRecordZeekStrings(d, "S1")
	require.NoError(t, err)
	cols := []zng.Column{{"ts", zng.TypeTime}, {"s2", zng.TypeString}}
	ts, _ := nano.Parse([]byte("123.456"))
	r, err = tab.AddColumns(r, cols, []zng.Value{zng.NewTime(ts), zng.NewString("S2")})
	require.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "S1", r.Value(0).String())
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "S2", r.Value(2).String())
	assert.Nil(t, r.Slice(4))
}
