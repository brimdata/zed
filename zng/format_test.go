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
	recType := zctx.LookupTypeRecord([]zng.Column{
		{"b", zng.TypeBstring},
		{"s", zng.TypeString},
	})

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
				{zng.OutFormatZeek, "foo"},
				{zng.OutFormatZeekAscii, "foo"},
				{zng.OutFormatZNG, "foo"},
			},
		},

		// An unset string is represented as -
		{
			zng.Value{zng.TypeBstring, nil},
			[]Expect{
				{zng.OutFormatZeek, "-"},
				{zng.OutFormatZeekAscii, "-"},
				{zng.OutFormatZNG, "-"},
			},
		},

		// A value consisting of just - must be escaped
		{
			zng.NewBstring("-"),
			[]Expect{
				{zng.OutFormatZeek, `\x2d`},
				{zng.OutFormatZeekAscii, `\x2d`},
				{zng.OutFormatZNG, `\x2d`},
			},
		},

		// A longer value containing - is not escaped
		{
			zng.NewBstring("--"),
			[]Expect{
				{zng.OutFormatZeek, "--"},
				{zng.OutFormatZeekAscii, "--"},
				{zng.OutFormatZNG, "--"},
			},
		},

		// Invalid UTF-8 is escaped
		{
			zng.Value{zng.TypeBstring, []byte{0xae, 0x8c, 0x9f, 0xf0}},
			[]Expect{
				{zng.OutFormatZeek, `\xae\x8c\x9f\xf0`},
				{zng.OutFormatZeekAscii, `\xae\x8c\x9f\xf0`},
				{zng.OutFormatZNG, `\xae\x8c\x9f\xf0`},
			},
		},

		// A backslash is escaped
		{
			zng.NewBstring(`\`),
			[]Expect{
				{zng.OutFormatZeek, `\\`},
				{zng.OutFormatZeekAscii, `\\`},
				{zng.OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zng.NewBstring("\n\t"),
			[]Expect{
				{zng.OutFormatZeek, `\x0a\x09`},
				{zng.OutFormatZeekAscii, `\x0a\x09`},
				{zng.OutFormatZNG, `\x0a\x09`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zng.NewBstring("a,b"),
			[]Expect{
				{zng.OutFormatZeek, `a,b`},
				{zng.OutFormatZeekAscii, `a,b`},
				{zng.OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zng.NewBstring("[hello"),
			[]Expect{
				{zng.OutFormatZeek, `[hello`},
				{zng.OutFormatZeekAscii, `[hello`},
				{zng.OutFormatZNG, `\x5bhello`},
			},
		},

		// Square bracket not at the start of a value is not escaped
		{
			zng.NewBstring("hello["),
			[]Expect{
				{zng.OutFormatZeek, `hello[`},
				{zng.OutFormatZeekAscii, `hello[`},
				{zng.OutFormatZNG, `hello[`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zng.NewBstring(";"),
			[]Expect{
				{zng.OutFormatZeek, `;`},
				{zng.OutFormatZeekAscii, `;`},
				{zng.OutFormatZNG, `\x3b`},
			},
		},

		// A non-ascii unicode code point is escaped in zeek-ascii
		// but left intact in other formats.
		{
			zng.NewBstring("🌮"),
			[]Expect{
				{zng.OutFormatZeek, "🌮"},
				{zng.OutFormatZeekAscii, `\xf0\x9f\x8c\xae`},
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
				{zng.OutFormatZeek, `\u002d`},
				{zng.OutFormatZeekAscii, `\u002d`},
				{zng.OutFormatZNG, `\u002d`},
			},
		},

		// A backslash is escaped
		{
			zng.NewString(`\`),
			[]Expect{
				{zng.OutFormatZeek, `\\`},
				{zng.OutFormatZeekAscii, `\\`},
				{zng.OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zng.NewString("\n\t"),
			[]Expect{
				{zng.OutFormatZeek, `\u{a}\u{9}`},
				{zng.OutFormatZeekAscii, `\u{a}\u{9}`},
				{zng.OutFormatZNG, `\u{a}\u{9}`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zng.NewString("a,b"),
			[]Expect{
				{zng.OutFormatZeek, `a,b`},
				{zng.OutFormatZeekAscii, `a,b`},
				{zng.OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zng.NewString("[hello"),
			[]Expect{
				{zng.OutFormatZeek, `[hello`},
				{zng.OutFormatZeekAscii, `[hello`},
				{zng.OutFormatZNG, `\u{5b}hello`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zng.NewString(";"),
			[]Expect{
				{zng.OutFormatZeek, `;`},
				{zng.OutFormatZeekAscii, `;`},
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
				{zng.OutFormatZeek, "-"},
				{zng.OutFormatZeekAscii, "-"},
				{zng.OutFormatZNG, "-"},
			},
		},

		// empty set
		{
			zng.Value{bstringSetType, []byte{}},
			[]Expect{
				{zng.OutFormatZeek, "(empty)"},
				{zng.OutFormatZeekAscii, "(empty)"},
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
				{zng.OutFormatZeek, "abc,xyz"},
				{zng.OutFormatZeekAscii, "abc,xyz"},
				{zng.OutFormatZNG, "[abc;xyz;]"},
			},
		},

		// A comma inside a string inside a set is escaped in Zeek.
		{
			zng.Value{bstringSetType, makeContainer([]byte("a,b"))},
			[]Expect{
				{zng.OutFormatZeek, `a\x2cb`},
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
				// not representable in zeek
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
				{zng.OutFormatZeek, "-"},
				{zng.OutFormatZeekAscii, "-"},
				{zng.OutFormatZNG, "-"},
			},
		},

		// empty vector
		{
			zng.Value{bstringVecType, []byte{}},
			[]Expect{
				{zng.OutFormatZeek, "(empty)"},
				{zng.OutFormatZeekAscii, "(empty)"},
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
				{zng.OutFormatZeek, "abc,xyz"},
				{zng.OutFormatZeekAscii, "abc,xyz"},
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
				// not representable in zeek
				{zng.OutFormatZNG, `[[a;b;];[x;y;];]`},
			},
		},

		// A comma inside a string inside a vector is escaped in Zeek.
		{
			zng.Value{bstringVecType, makeContainer([]byte("a,b"))},
			[]Expect{
				{zng.OutFormatZeek, `a\x2cb`},
			},
		},

		// vector containing unset
		{
			zng.Value{
				bstringVecType,
				makeContainer([]byte("-"), nil),
			},
			[]Expect{
				{zng.OutFormatZeek, `\x2d,-`},
				{zng.OutFormatZNG, `[\x2d;-;]`},
			},
		},

		// vector containing empty string
		{
			zng.Value{bstringVecType, makeContainer([]byte{})},
			[]Expect{
				{zng.OutFormatZeek, ""},
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
