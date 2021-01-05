package filter

import (
	"testing"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

func TestFieldNameIter(t *testing.T) {
	const typeString = "record[r1:record[r2:record[s:string,r3:record[t:time]],a:array[int64],r4:record[i:ip]],empty:record[]]"
	typ, err := resolver.NewContext().LookupByName(typeString)
	require.NoError(t, err)
	var f fieldNameIter
	f.init(typ.(*zng.TypeRecord))
	require.False(t, f.done())
	require.Exactly(t, "r1.r2.s", string(f.next()))
	require.False(t, f.done())
	require.Exactly(t, "r1.r2.r3.t", string(f.next()))
	require.False(t, f.done())
	require.Exactly(t, "r1.a", string(f.next()))
	require.False(t, f.done())
	require.Exactly(t, "r1.r4.i", string(f.next()))
	require.False(t, f.done())
	require.Exactly(t, "empty", string(f.next()))
	require.True(t, f.done())
}
func TestFieldNameIterEmptyTopLevelRecord(t *testing.T) {
	typ, err := resolver.NewContext().LookupByName("record[]")
	require.NoError(t, err)
	var f fieldNameIter
	f.init(typ.(*zng.TypeRecord))
	require.True(t, f.done())
}
