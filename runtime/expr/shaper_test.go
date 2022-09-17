package expr

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/stretchr/testify/assert"
)

func TestBestUnionTag(t *testing.T) {
	u8 := zed.TypeUint8
	zctx := zed.NewContext()
	u8named1 := zctx.LookupTypeNamed("u8named1", u8)
	u8named2 := zctx.LookupTypeNamed("u8named2", u8)
	u8named3 := zctx.LookupTypeNamed("u8named3", u8)

	assert.Equal(t, -1, bestUnionTag(u8, nil))
	assert.Equal(t, -1, bestUnionTag(u8, u8))
	assert.Equal(t, -1, bestUnionTag(zed.TypeUint16, zctx.LookupTypeUnion([]zed.Type{u8})))

	test := func(expected, needle zed.Type, haystack []zed.Type) {
		t.Helper()
		union := zctx.LookupTypeUnion(haystack)
		typ, err := union.Type(bestUnionTag(needle, union))
		if assert.NoError(t, err) {
			assert.Equal(t, expected, typ)
		}

	}

	// Needle is in haystack.
	test(u8, u8, []zed.Type{u8, u8named1, u8named2})
	test(u8, u8, []zed.Type{u8named2, u8named1, u8})
	test(u8, u8, []zed.Type{u8named1, u8, u8named2})
	test(u8named2, u8named2, []zed.Type{u8, u8named1, u8named2})
	test(u8named2, u8named2, []zed.Type{u8named2, u8named1, u8})
	test(u8named2, u8named2, []zed.Type{u8, u8named2, u8named1})

	// Underlying type of needle is in haystack.
	test(u8, u8named1, []zed.Type{u8, u8named2, u8named3})
	test(u8, u8named1, []zed.Type{u8named3, u8named2, u8})
	test(u8, u8named1, []zed.Type{u8named2, u8, u8named3})

	// Type compatible with needle is in haystack.
	test(u8named1, u8, []zed.Type{u8named1, u8named2, u8named3})
	test(u8named3, u8named1, []zed.Type{u8named3, u8named2})
}
