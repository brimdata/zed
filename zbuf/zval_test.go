package zbuf

import (
	"testing"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
)

func TestEscapes(t *testing.T) {
	zctx := resolver.NewContext()
	cases := []struct {
		typ      zng.Type
		zv       zcode.Bytes
		expected string
	}{
		{zng.TypeBstring, []byte("foo"), "foo"},
		{zng.TypeBstring, nil, "-"},
		{zng.TypeBstring, []byte("-"), "\\x2d"},
		{
			zctx.LookupTypeVector(zng.TypeBstring),
			zcode.AppendPrimitive(zcode.AppendPrimitive(nil, []byte("-")), nil),
			"\\x2d,-",
		},
	}
	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			v := zng.Value{tc.typ, tc.zv}
			res := v.FormatAs(zng.OUT_FORMAT_ZEEK)
			assert.Equal(t, tc.expected, res)
		})
	}
}
