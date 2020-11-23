package zng_test

import (
	"testing"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
)

func makeContainer(vals ...[]byte) zcode.Bytes {
	var zv zcode.Bytes
	for _, v := range vals {
		zv = zcode.AppendPrimitive(zv, v)
	}
	return zv
}

func TestFormatting(t *testing.T) {
	zctx := resolver.NewContext()
	bstringSetType := zctx.LookupTypeSet(zng.TypeBstring)
	bstringVecType := zctx.LookupTypeArray(zng.TypeBstring)
	setOfVectorsType := zctx.LookupTypeSet(bstringVecType)
	vecOfVectorsType := zctx.LookupTypeArray(bstringVecType)
	recType, err := zctx.LookupTypeRecord([]zng.Column{
		{"b", zng.TypeBstring},
		{"s", zng.TypeString},
	})
	assert.NoError(t, err)

	type Expect struct {
		fmt      zng.OutFmt
		expected string
	}

	cases := []struct {
		val      zng.Value
		expected []Expect
	}{
		//
		// Test bstrings
		//

		// An ascii string
		{
			zng.NewBstring("foo"),
			[]Expect{
				{zng.OutFormatZNG, "foo"},
			},
		},

		// An unset string is represented as -
		{
			zng.Value{zng.TypeBstring, nil},
			[]Expect{
				{zng.OutFormatZNG, "-"},
			},
		},

		// A value consisting of just - must be escaped
		{
			zng.NewBstring("-"),
			[]Expect{
				{zng.OutFormatZNG, `\x2d`},
			},
		},

		// A longer value containing - is not escaped
		{
			zng.NewBstring("--"),
			[]Expect{
				{zng.OutFormatZNG, "--"},
			},
		},

		// Invalid UTF-8 is escaped
		{
			zng.Value{zng.TypeBstring, []byte{0xae, 0x8c, 0x9f, 0xf0}},
			[]Expect{
				{zng.OutFormatZNG, `\xae\x8c\x9f\xf0`},
			},
		},

		// A backslash is escaped
		{
			zng.NewBstring(`\`),
			[]Expect{
				{zng.OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zng.NewBstring("\n\t"),
			[]Expect{
				{zng.OutFormatZNG, `\x0a\x09`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zng.NewBstring("a,b"),
			[]Expect{
				{zng.OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zng.NewBstring("[hello"),
			[]Expect{
				{zng.OutFormatZNG, `\x5bhello`},
			},
		},

		// Square bracket not at the start of a value is not escaped
		{
			zng.NewBstring("hello["),
			[]Expect{
				{zng.OutFormatZNG, `hello[`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zng.NewBstring(";"),
			[]Expect{
				{zng.OutFormatZNG, `\x3b`},
			},
		},

		// A non-ascii unicode code point is left intact.
		{
			zng.NewBstring("🌮"),
			[]Expect{
				{zng.OutFormatZNG, "🌮"},
			},
		},

		//
		// Test string escapes (\u vs \x)
		//

		// A value consisting of just - must be escaped
		{
			zng.NewString("-"),
			[]Expect{
				{zng.OutFormatZNG, `\u002d`},
			},
		},

		// A backslash is escaped
		{
			zng.NewString(`\`),
			[]Expect{
				{zng.OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zng.NewString("\n\t"),
			[]Expect{
				{zng.OutFormatZNG, `\u{a}\u{9}`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zng.NewString("a,b"),
			[]Expect{
				{zng.OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zng.NewString("[hello"),
			[]Expect{
				{zng.OutFormatZNG, `\u{5b}hello`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zng.NewString(";"),
			[]Expect{
				{zng.OutFormatZNG, `\u{3b}`},
			},
		},

		//
		// Test sets
		//

		// unset set
		{
			zng.Value{bstringSetType, nil},
			[]Expect{
				{zng.OutFormatZNG, "-"},
			},
		},

		// empty set
		{
			zng.Value{bstringSetType, []byte{}},
			[]Expect{
				{zng.OutFormatZNG, "[]"},
			},
		},

		// simple set
		{
			zng.Value{
				bstringSetType,
				makeContainer([]byte("abc"), []byte("xyz")),
			},
			[]Expect{
				{zng.OutFormatZNG, "[abc;xyz;]"},
			},
		},

		// set containing vectors
		{
			zng.Value{
				setOfVectorsType,
				makeContainer(
					makeContainer([]byte("a"), []byte("b")),
					makeContainer([]byte("x"), []byte("y")),
				),
			},
			[]Expect{
				{zng.OutFormatZNG, `[[a;b;];[x;y;];]`},
			},
		},

		//
		// Test vectors
		//

		// unset vector
		{
			zng.Value{bstringVecType, nil},
			[]Expect{
				{zng.OutFormatZNG, "-"},
			},
		},

		// empty vector
		{
			zng.Value{bstringVecType, []byte{}},
			[]Expect{
				{zng.OutFormatZNG, "[]"},
			},
		},

		// simple vector
		{
			zng.Value{
				bstringVecType,
				makeContainer([]byte("abc"), []byte("xyz")),
			},
			[]Expect{
				{zng.OutFormatZNG, "[abc;xyz;]"},
			},
		},

		// vector containing vectors
		{
			zng.Value{
				vecOfVectorsType,
				makeContainer(
					makeContainer([]byte("a"), []byte("b")),
					makeContainer([]byte("x"), []byte("y")),
				),
			},
			[]Expect{
				{zng.OutFormatZNG, `[[a;b;];[x;y;];]`},
			},
		},

		// vector containing unset
		{
			zng.Value{
				bstringVecType,
				makeContainer([]byte("-"), nil),
			},
			[]Expect{
				{zng.OutFormatZNG, `[\x2d;-;]`},
			},
		},

		// vector containing empty string
		{
			zng.Value{bstringVecType, makeContainer([]byte{})},
			[]Expect{
				{zng.OutFormatZNG, `[;]`},
			},
		},

		//
		// Test records
		//

		// Simple record
		{
			zng.Value{
				recType,
				makeContainer([]byte("foo"), []byte("bar")),
			},
			[]Expect{
				{zng.OutFormatZNG, `[foo;bar;]`},
			},
		},

		// Record with nils
		{
			zng.Value{recType, makeContainer(nil, nil)},
			[]Expect{
				{zng.OutFormatZNG, `[-;-;]`},
			},
		},
	}
	for _, tc := range cases {
		for _, expect := range tc.expected {
			t.Run(expect.expected, func(t *testing.T) {
				res := tc.val.Format(expect.fmt)
				assert.Equal(t, expect.expected, res)
			})
		}
	}
}
