package resolver

import (
	"testing"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableAddColumns(t *testing.T) {
	tab := NewTable()
	d := tab.GetByColumns([]zeek.Column{{"s1", zeek.TypeString}})
	r, err := zng.NewRecordZeekStrings(d, "S1")
	require.NoError(t, err)
	cols := []zeek.Column{{"ts", zeek.TypeTime}, {"s2", zeek.TypeString}}
	ts, _ := nano.Parse([]byte("123.456"))
	r, err = tab.AddColumns(r, cols, []zeek.Value{zeek.NewTime(ts), zeek.NewString("S2")})
	require.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "S1", r.Value(0).String())
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "S2", r.Value(2).String())
	assert.Nil(t, r.Slice(4))
}
