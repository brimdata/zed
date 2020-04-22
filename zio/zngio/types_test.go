package zngio_test

import (
	"bytes"
	"testing"

	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextSerialization(t *testing.T) {
	ctx := resolver.NewContext()
	ctx.LookupByName("record[a:set[string],b:int32]]")
	ctx.LookupByName("record[a:array[record[a:int32,b:int32]],b:int32]]")
	b, _ := ctx.Serialize()
	reader := bytes.NewReader(b)
	newCtx := resolver.NewContext()
	err := zngio.ReadTypeContext(reader, newCtx)
	require.NoError(t, err)
	r1, err := ctx.LookupByName("record[a:int32,b:int32]")
	require.NoError(t, err)
	r2, err := newCtx.LookupByName("record[a:int32,b:int32]")
	require.NoError(t, err)
	assert.EqualValues(t, r1.ID(), r2.ID())
}
