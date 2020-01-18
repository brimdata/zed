package zbuf

import (
	"testing"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
)

func TestZvalToZeekString(t *testing.T) {
	zctx := resolver.NewContext()
	cases := []struct {
		typ      zng.Type
		zv       zcode.Bytes
		expected string
	}{
		{zng.TypeString, []byte("foo"), "foo"},
		{zng.TypeString, nil, "-"},
		{zng.TypeString, []byte("-"), "\\x2d"},
		{
			zctx.LookupVectorType(zng.TypeString),
			zcode.AppendPrimitive(zcode.AppendPrimitive(nil, []byte("-")), nil),
			"\\x2d,-",
		},
	}
	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			res := ZvalToZeekString(tc.typ, tc.zv, true)
			assert.Equal(t, tc.expected, res)
		})
	}
}
