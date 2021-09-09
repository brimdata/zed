package expr

import (
	"testing"

	"github.com/brimdata/zed/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBestUnionSelector(t *testing.T) {
	u8 := zng.TypeUint8
	zctx := zng.NewContext()
	u8alias1, err := zctx.LookupTypeAlias("u8alias1", u8)
	require.NoError(t, err)
	u8alias2, err := zctx.LookupTypeAlias("u8alias2", u8)
	require.NoError(t, err)
	u8alias3, err := zctx.LookupTypeAlias("u8alias3", u8)
	require.NoError(t, err)

	assert.Equal(t, -1, bestUnionSelector(u8, nil))
	assert.Equal(t, -1, bestUnionSelector(u8, u8))
	assert.Equal(t, -1, bestUnionSelector(zng.TypeUint16, zctx.LookupTypeUnion([]zng.Type{u8})))

	test := func(expected, needle zng.Type, haystack []zng.Type) {
		t.Helper()
		union := zctx.LookupTypeUnion(haystack)
		typ, err := union.Type(bestUnionSelector(needle, union))
		if assert.NoError(t, err) {
			assert.Equal(t, expected, typ)
		}

	}

	// Needle is in haystack.
	test(u8, u8, []zng.Type{u8, u8alias1, u8alias2})
	test(u8, u8, []zng.Type{u8alias2, u8alias1, u8})
	test(u8, u8, []zng.Type{u8alias1, u8, u8alias2})
	test(u8alias2, u8alias2, []zng.Type{u8, u8alias1, u8alias2})
	test(u8alias2, u8alias2, []zng.Type{u8alias2, u8alias1, u8})
	test(u8alias2, u8alias2, []zng.Type{u8, u8alias2, u8alias1})

	// Underlying type of needle is in haystack.
	test(u8, u8alias1, []zng.Type{u8, u8alias2, u8alias3})
	test(u8, u8alias1, []zng.Type{u8alias3, u8alias2, u8})
	test(u8, u8alias1, []zng.Type{u8alias2, u8, u8alias3})

	// Type compatible with needle is in haystack.
	test(u8alias1, u8, []zng.Type{u8alias1, u8alias2, u8alias3})
	test(u8alias3, u8alias1, []zng.Type{u8alias3, u8alias2})
}
