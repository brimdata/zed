package filter

import (
	"testing"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

func TestFieldNameIter(t *testing.T) {
	const typeString = "record[r1:record[r2:record[s:string,r3:record[t:time]],a:array[int64],r4:record[i:ip]]]"
	typ, err := resolver.NewContext().LookupByName(typeString)
	require.NoError(t, err)
	f := newFieldNameIter(typ.(*zng.TypeRecord))
	require.Exactly(t, "r1.r2.s", f.next())
	require.False(t, f.done())
	require.Exactly(t, "r1.r2.r3.t", f.next())
	require.False(t, f.done())
	require.Exactly(t, "r1.a", f.next())
	require.False(t, f.done())
	require.Exactly(t, "r1.r4.i", f.next())
	require.True(t, f.done())
}
