package zed

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextLookupTypeNamedErrors(t *testing.T) {
	zctx := NewContext()

	_, err := zctx.LookupTypeNamed("\xff", TypeNull)
	assert.EqualError(t, err, `bad type name "\xff": invalid UTF-8`)

	_, err = zctx.LookupTypeNamed("null", TypeNull)
	assert.EqualError(t, err, `bad type name "null": primitive type name`)
}

func TestContextLookupTypeNamedAndLookupTypeDef(t *testing.T) {
	zctx := NewContext()

	assert.Nil(t, zctx.LookupTypeDef("x"))

	named1, err := zctx.LookupTypeNamed("x", TypeNull)
	require.NoError(t, err)
	assert.Same(t, named1, zctx.LookupTypeDef("x"))

	named2, err := zctx.LookupTypeNamed("x", TypeInt8)
	require.NoError(t, err)
	assert.Same(t, named2, zctx.LookupTypeDef("x"))

	named3, err := zctx.LookupTypeNamed("x", TypeNull)
	require.NoError(t, err)
	assert.Same(t, named3, zctx.LookupTypeDef("x"))
	assert.Same(t, named3, named1)
}
