package zed_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextLookupTypeNamedErrors(t *testing.T) {
	zctx := zed.NewContext()

	_, err := zctx.LookupTypeNamed("\xff", zed.TypeNull)
	assert.EqualError(t, err, `bad type name "\xff": invalid UTF-8`)

	_, err = zctx.LookupTypeNamed("null", zed.TypeNull)
	assert.EqualError(t, err, `bad type name "null": primitive type name`)
}

func TestContextLookupTypeNamedAndLookupTypeDef(t *testing.T) {
	zctx := zed.NewContext()

	assert.Nil(t, zctx.LookupTypeDef("x"))

	named1, err := zctx.LookupTypeNamed("x", zed.TypeNull)
	require.NoError(t, err)
	assert.Same(t, named1, zctx.LookupTypeDef("x"))

	named2, err := zctx.LookupTypeNamed("x", zed.TypeInt8)
	require.NoError(t, err)
	assert.Same(t, named2, zctx.LookupTypeDef("x"))

	named3, err := zctx.LookupTypeNamed("x", zed.TypeNull)
	require.NoError(t, err)
	assert.Same(t, named3, zctx.LookupTypeDef("x"))
	assert.Same(t, named3, named1)
}

func TestContextTranslateTypeNameConflictUnion(t *testing.T) {
	// This test confirms that a union with complicated type renaming is properly
	// decoded.  There was a bug where child typedefs would override the
	// top level typedef in TranslateType so foo in the value below had
	// two of the same union type instead of the two it should have had.
	zctx := zed.NewContext()
	val := zson.MustParseValue(zctx, `[{x:{y:63}}(=foo),{x:{abcdef:{x:{y:127}}(foo)}}(=foo)]`)
	foreign := zed.NewContext()
	twin, err := foreign.TranslateType(val.Type)
	require.NoError(t, err)
	union := twin.(*zed.TypeArray).Type.(*zed.TypeUnion)
	assert.Equal(t, `foo={x:{abcdef:foo={x:{y:int64}}}}`, zson.String(union.Types[0]))
	assert.Equal(t, `foo={x:{y:int64}}`, zson.String(union.Types[1]))
}
