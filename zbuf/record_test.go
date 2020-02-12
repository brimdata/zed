package zbuf

import (
	"testing"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lookup(r *zng.Record, col int) zcode.Bytes {
	zv, _ := r.Slice(col)
	return zv
}

func TestNewRecordZeekStrings(t *testing.T) {
	zctx := resolver.NewContext()
	typ, err := zctx.LookupByName("record[_path:string,ts:time,data:string]")
	require.NoError(t, err)
	d := typ.(*zng.TypeRecord)

	_, err = NewRecordZeekStrings(d, "some path", "123.456")
	assert.EqualError(t, err, "got 2 values, expected 3")

	_, err = NewRecordZeekStrings(d, "some path", "123.456", "some data", "unexpected")
	assert.EqualError(t, err, "got 4 values, expected 3")

	r, err := NewRecordZeekStrings(d, "some path", "123.4567", "some data")
	assert.NoError(t, err)
	assert.EqualValues(t, 123456700000, r.Ts)
	s, _ := r.AccessString("_path")
	assert.EqualValues(t, "some path", s)
	assert.EqualValues(t, "123.4567", r.Value(1).String())
	assert.EqualValues(t, "some data", lookup(r, 2))
	assert.Nil(t, lookup(r, 3))

	r, err = NewRecordZeekStrings(d, "some path", "123.456", "")
	assert.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", lookup(r, 0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "", lookup(r, 2))
	assert.Nil(t, lookup(r, 3))
}

func zs(ss ...string) [][]byte {
	var vals [][]byte
	for _, s := range ss {
		b := []byte(s)
		vals = append(vals, b)
	}
	return vals
}

func res(v ...string) []string {
	var out []string
	for _, s := range v {
		out = append(out, s)
	}
	return out
}

func encode(typ *zng.TypeRecord, vals [][]byte) (zcode.Bytes, error) {
	zv, _, err := NewRawAndTsFromZeekValues(typ, -1, vals)
	return zv, err
}

func TestEncodeZeekStrings(t *testing.T) {
	zctx := resolver.NewContext()
	typ, err := zctx.LookupByName("record[_path:string,ts:time,data:string]")
	require.NoError(t, err)
	d := typ.(*zng.TypeRecord)

	_, err = encode(d, zs("some path", "123.456"))
	assert.EqualError(t, err, "got 2 values, expected 3")

	_, err = encode(d, zs("some path", "123.456", "some data", "unexpected"))
	assert.EqualError(t, err, "got 4 values, expected 3")

	zv, err := encode(d, zs("some path", "123.456", "some data"))
	assert.NoError(t, err)
	r, err := zng.NewRecord(d, zv)
	assert.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", lookup(r, 0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "some data", lookup(r, 2))
	assert.Nil(t, lookup(r, 3))

	zv, err = encode(d, zs("some path", "123.456", ""))
	assert.NoError(t, err)
	r, err = zng.NewRecord(d, zv)
	assert.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", lookup(r, 0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "", lookup(r, 2))
	assert.Nil(t, lookup(r, 3))

	zv, err = encode(d, zs("some path", "123.456", "-"))
	assert.NoError(t, err)
	r, err = zng.NewRecord(d, zv)
	assert.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", lookup(r, 0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, zcode.Bytes(nil), lookup(r, 2))
	assert.Nil(t, lookup(r, 3))

	typ, err = zctx.LookupByName("record[_path:string,ts:time,data:set[int32]]")
	require.NoError(t, err)
	d = typ.(*zng.TypeRecord)

	cases := []struct {
		input    [][]byte
		expected []string
	}{
		{zs("some path", "123.456", "-"), res("some path", "123.456", "-")},
		{zs("some path", "123.456", "(empty)"), res("some path", "123.456", "set[]")},
		// XXX this is an error
		//{zs("some path", "123.456", ""), zvals(z("some path"), z("123.456"), z(xxx)},
		{zs("some path", "123.456", "987"), res("some path", "123.456", "set[987]")},
		{zs("some path", "123.456", "987,65"), res("some path", "123.456", "set[987,65]")},
	}
	for i, c := range cases {
		zv, err := encode(d, c.input)
		assert.NoError(t, err)
		r, err := zng.NewRecord(d, zv)
		assert.NoError(t, err)
		assert.EqualValues(t, 123456000000, r.Ts)
		for j, e := range c.expected {
			assert.EqualValues(t, e, r.Value(j).String(), "case %d, index: %d", i, j)
		}
	}
}
