package zbuf

import (
	"testing"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRecordZeekStrings(t *testing.T) {
	typ, err := zng.LookupType("record[_path:string,ts:time,data:string]")
	require.NoError(t, err)
	d := NewDescriptor(typ.(*zng.TypeRecord))

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
	assert.EqualValues(t, "some data", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	r, err = NewRecordZeekStrings(d, "some path", "123.456", "")
	assert.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "", r.Slice(2))
	assert.Nil(t, r.Slice(3))
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

func encode(d *Descriptor, vals [][]byte) (zcode.Bytes, error) {
	zv, _, err := NewRawAndTsFromZeekValues(d, -1, vals)
	return zv, err
}

func TestEncodeZeekStrings(t *testing.T) {
	typ, err := zng.LookupType("record[_path:string,ts:time,data:string]")
	require.NoError(t, err)
	d := NewDescriptor(typ.(*zng.TypeRecord))

	_, err = encode(d, zs("some path", "123.456"))
	assert.EqualError(t, err, "got 2 values, expected 3")

	_, err = encode(d, zs("some path", "123.456", "some data", "unexpected"))
	assert.EqualError(t, err, "got 4 values, expected 3")

	zv, err := encode(d, zs("some path", "123.456", "some data"))
	assert.NoError(t, err)
	r := NewRecordNoTs(d, zv)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "some data", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	zv, err = encode(d, zs("some path", "123.456", ""))
	assert.NoError(t, err)
	r = NewRecordNoTs(d, zv)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, "", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	zv, err = encode(d, zs("some path", "123.456", "-"))
	assert.NoError(t, err)
	r = NewRecordNoTs(d, zv)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Value(1).String())
	assert.EqualValues(t, zcode.Bytes(nil), r.Slice(2))
	assert.Nil(t, r.Slice(3))

	typ, err = zng.LookupType("record[_path:string,ts:time,data:set[int]]")
	require.NoError(t, err)
	d = NewDescriptor(typ.(*zng.TypeRecord))

	cases := []struct {
		input    [][]byte
		expected []string
	}{
		//XXX last arg should be "-" instead of set[]
		{zs("some path", "123.456", "-"), res("some path", "123.456", "set[]")},
		{zs("some path", "123.456", "(empty)"), res("some path", "123.456", "set[]")},
		// XXX this is an error
		//{zs("some path", "123.456", ""), zvals(z("some path"), z("123.456"), z(xxx)},
		{zs("some path", "123.456", "987"), res("some path", "123.456", "set[987]")},
		{zs("some path", "123.456", "987,65"), res("some path", "123.456", "set[987,65]")},
	}
	for i, c := range cases {
		zv, err := encode(d, c.input)
		assert.NoError(t, err)
		r := NewRecordNoTs(d, zv)
		assert.EqualValues(t, 123456000000, r.Ts)
		for j, e := range c.expected {
			assert.EqualValues(t, e, r.Value(j).String(), "case %d, index: %d", i, j)
		}
	}
}
