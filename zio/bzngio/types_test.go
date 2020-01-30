package bzngio_test

import (
	"bytes"
	"testing"

	"github.com/mccanne/zq/zio/bzngio"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextSerialization(t *testing.T) {
	ctx := resolver.NewContext()
	ctx.LookupByName("record[a:set[string],b:int]]")
	ctx.LookupByName("record[a:array[record[a:int,b:int]],b:int]]")
	b, _ := ctx.Serialize()
	reader := bytes.NewReader(b)
	newCtx := resolver.NewContext()
	err := bzngio.ReadTypeContext(reader, newCtx)
	require.NoError(t, err)
	r1, err := newCtx.LookupByName("record[a:int,b:int]")
	require.NoError(t, err)
	r2, err := newCtx.LookupByName("record[a:int,b:int]")
	require.NoError(t, err)
	assert.EqualValues(t, r1.ID(), r2.ID())
}
