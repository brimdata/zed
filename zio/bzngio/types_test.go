package bzngio_test

/* XXX
import (
	"bytes"
	"testing"

	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextSerialization(t *testing.T) {
	ctx := resolver.NewContext()
	ctx.LookupByName("record[a:set[string],b:int64]]")
	ctx.LookupByName("record[a:array[record[a:int64,b:int64]],b:int64]]")
	b, _ := ctx.Serialize()
	reader := bytes.NewReader(b)
	newCtx := resolver.NewContext()
	err := bzngio.ReadTypeContext(reader, newCtx)
	require.NoError(t, err)
	r1, err := newCtx.LookupByName("record[a:int64,b:int64]")
	require.NoError(t, err)
	r2, err := newCtx.LookupByName("record[a:int64,b:int64]")
	require.NoError(t, err)
	assert.EqualValues(t, r1.ID(), r2.ID())
}
*/
