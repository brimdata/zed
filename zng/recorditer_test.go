package zng_test

import (
	"strings"
	"testing"

	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"

	"github.com/stretchr/testify/require"
)

func parseZng(s string) (*zng.Record, error) {
	reader := zngio.NewReader(strings.NewReader(s), resolver.NewContext())
	return reader.Read()
}

func TestRecordIter(t *testing.T) {
	// Test a few edge cases: record with another record as the first
	// field, record with another record as the last field, non-record
	// container types inside records...
	rec, err := parseZng(`
#0:record[r1:record[r2:record[s:string],a:array[int64],r3:record[i:ip]]]
0:[[[hello;][1;2;3;][1.2.3.4;]]]`)
	require.NoError(t, err)

	it := rec.FieldIter()
	require.False(t, it.Done(), "iterator is not exhausted")

	name, val, err := it.Next()
	require.NoError(t, err)
	require.Equal(t, "r1.r2.s", name, "got correct field name")
	require.Equal(t, zng.TypeString, val.Type, "got correct type for first field")
	require.Equal(t, "hello", string(val.Bytes), "got correct value for first field")
	require.False(t, it.Done(), "iterator is not exhausted")

	name, val, err = it.Next()
	require.NoError(t, err)
	require.Equal(t, "r1.a", name, "got correct field name")
	l, err := val.ContainerLength()
	require.NoError(t, err)
	require.Equal(t, 3, l, "got array of length 3")
	require.False(t, it.Done(), "iterator is not exhausted")

	name, val, err = it.Next()
	require.NoError(t, err)
	require.Equal(t, "r1.r3.i", name, "got correct field name")
	require.Equal(t, zng.TypeIP, val.Type, "got correct type for last field")
	require.True(t, it.Done(), "iterator is exhausted after last field")
}
