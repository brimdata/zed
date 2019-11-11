package zson

import (
	"testing"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRecordZeekStrings(t *testing.T) {
	typ, err := zeek.LookupType("record[_path:string,ts:time,data:string]")
	require.NoError(t, err)
	d := NewDescriptor(typ.(*zeek.TypeRecord))

	_, err = NewRecordZeekStrings(d, "some path", "123.456")
	assert.EqualError(t, err, "got 2 values, expected 3")

	_, err = NewRecordZeekStrings(d, "some path", "123.456", "some data", "unexpected")
	assert.EqualError(t, err, "got 4 values, expected 3")

	r, err := NewRecordZeekStrings(d, "some path", "123.456", "some data")
	assert.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.EqualValues(t, "some data", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	r, err = NewRecordZeekStrings(d, "some path", "123.456", "")
	assert.NoError(t, err)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.EqualValues(t, "", r.Slice(2))
	assert.Nil(t, r.Slice(3))
}

func TestNewRawAndTsFromZeekValues(t *testing.T) {
	typ, err := zeek.LookupType("record[_path:string,ts:time,data:string]")
	require.NoError(t, err)
	d := NewDescriptor(typ.(*zeek.TypeRecord))

	b := func(s string) []byte { return []byte(s) }
	_, _, err = NewRawAndTsFromZeekValues(d, 1, [][]byte{b("some path"), b("123.456")})
	assert.EqualError(t, err, "got 2 values, expected 3")

	_, _, err = NewRawAndTsFromZeekValues(d, 1, [][]byte{b("some path"), b("123.456"), b("some data"), b("unexpected")})
	assert.EqualError(t, err, "got 4 values, expected 3")

	raw, ts, err := NewRawAndTsFromZeekValues(d, 1, [][]byte{b("some path"), b("123.456"), b("some data")})
	assert.NoError(t, err)
	r := NewRecord(d, ts, raw)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.EqualValues(t, "some data", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	raw, ts, err = NewRawAndTsFromZeekValues(d, 1, [][]byte{b("some path"), b("123.456"), b("")})
	assert.NoError(t, err)
	r = NewRecord(d, ts, raw)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.EqualValues(t, "", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	raw, ts, err = NewRawAndTsFromZeekValues(d, 1, [][]byte{b("some path"), b("123.456"), b("-")})
	assert.NoError(t, err)
	r = NewRecord(d, ts, raw)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.Nil(t, r.Slice(2))
	assert.Nil(t, r.Slice(3))

	typ, err = zeek.LookupType("record[_path:string,ts:time,data:set[int]]")
	require.NoError(t, err)
	d = NewDescriptor(typ.(*zeek.TypeRecord))

	cases := []struct {
		input    []string
		expected [][]byte
	}{
		{[]string{"some path", "123.456", "-"}, [][]byte{b("some path"), b("123.456"), nil, nil}},
		{[]string{"some path", "123.456", "(empty)"}, [][]byte{b("some path"), b("123.456"), b(""), nil}},
		{[]string{"some path", "123.456", ""}, [][]byte{b("some path"), b("123.456"), b("\x01"), nil}},
		{[]string{"some path", "123.456", "987"}, [][]byte{b("some path"), b("123.456"), b("\x04987"), nil}},
		{[]string{"some path", "123.456", "987,65"}, [][]byte{b("some path"), b("123.456"), b("\x04987" + "\x0365"), nil}},
	}
	for i, c := range cases {
		var bs [][]byte
		for _, s := range c.input {
			bs = append(bs, b(s))
		}
		raw, ts, err := NewRawAndTsFromZeekValues(d, 1, bs)
		assert.NoError(t, err)
		r := NewRecord(d, ts, raw)
		assert.EqualValues(t, 123456000000, r.Ts)
		for j, e := range c.expected {
			assert.EqualValues(t, e, r.Slice(j), "case %d, index: %d", i, j)
		}
	}
}
