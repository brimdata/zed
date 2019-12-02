package zson

import (
	"testing"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
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

func zs(ss ...string) [][]byte {
	var vals [][]byte
	for _, s := range ss {
		b := []byte(s)
		vals = append(vals, b)
	}
	return vals
}

func zvals(zvs ...zval.Encoding) []zval.Encoding {
	var vals []zval.Encoding
	for _, zv := range zvs {
		vals = append(vals, zv)
	}
	return vals
}

func esc(s string) []byte {
	return []byte(zeek.Escape([]byte(s)))

}
func z(s string) zval.Encoding {
	return []byte(s)
}

func zc(ss ...string) zval.Encoding {
	var zv zval.Encoding
	for _, s := range ss {
		zv = zval.AppendValue(zv, esc(s))
	}
	return zv
}

func zempty() zval.Encoding {
	return make(zval.Encoding, 0)
}

func zunset() zval.Encoding {
	return nil
}

func zunsetc() zval.Encoding {
	return nil
}

func encode(d *Descriptor, vals [][]byte) (zval.Encoding, error) {
	zv, _, err := NewRawAndTsFromZeekValues(d, -1, vals)
	return zv, err
}

func TestEncodeZeekStrings(t *testing.T) {
	typ, err := zeek.LookupType("record[_path:string,ts:time,data:string]")
	require.NoError(t, err)
	d := NewDescriptor(typ.(*zeek.TypeRecord))

	_, err = encode(d, zs("some path", "123.456"))
	assert.EqualError(t, err, "got 2 values, expected 3")

	_, err = encode(d, zs("some path", "123.456", "some data", "unexpected"))
	assert.EqualError(t, err, "got 4 values, expected 3")

	zv, err := encode(d, zs("some path", "123.456", "some data"))
	assert.NoError(t, err)
	r := NewRecordNoTs(d, zv)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.EqualValues(t, "some data", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	zv, err = encode(d, zs("some path", "123.456", ""))
	assert.NoError(t, err)
	r = NewRecordNoTs(d, zv)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.EqualValues(t, "", r.Slice(2))
	assert.Nil(t, r.Slice(3))

	zv, err = encode(d, zs("some path", "123.456", "-"))
	assert.NoError(t, err)
	r = NewRecordNoTs(d, zv)
	assert.EqualValues(t, 123456000000, r.Ts)
	assert.EqualValues(t, "some path", r.Slice(0))
	assert.EqualValues(t, "123.456", r.Slice(1))
	assert.EqualValues(t, zval.Encoding(nil), r.Slice(2))
	assert.Nil(t, r.Slice(3))

	typ, err = zeek.LookupType("record[_path:string,ts:time,data:set[int]]")
	require.NoError(t, err)
	d = NewDescriptor(typ.(*zeek.TypeRecord))

	cases := []struct {
		input    [][]byte
		expected []zval.Encoding
	}{
		{zs("some path", "123.456", "-"), zvals(z("some path"), z("123.456"), zunsetc())},
		{zs("some path", "123.456", "(empty)"), zvals(z("some path"), z("123.456"), zempty())},
		// XXX this is an error
		//{zs("some path", "123.456", ""), zvals(z("some path"), z("123.456"), z(xxx)},
		{zs("some path", "123.456", "987"), zvals(z("some path"), z("123.456"), zc("987"))},
		{zs("some path", "123.456", "987,65"), zvals(z("some path"), z("123.456"), zc("987", "65"))},
	}
	for i, c := range cases {
		zv, err := encode(d, c.input)
		assert.NoError(t, err)
		r := NewRecordNoTs(d, zv)
		assert.EqualValues(t, 123456000000, r.Ts)
		for j, e := range c.expected {
			assert.EqualValues(t, e, r.Slice(j), "case %d, index: %d", i, j)
		}
	}
}
