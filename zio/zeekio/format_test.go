package zeekio

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
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
	zctx := zed.NewContext()
	bstringSetType := zctx.LookupTypeSet(zed.TypeBstring)
	bstringVecType := zctx.LookupTypeArray(zed.TypeBstring)
	setOfVectorsType := zctx.LookupTypeSet(bstringVecType)
	vecOfVectorsType := zctx.LookupTypeArray(bstringVecType)
	recType, err := zctx.LookupTypeRecord([]zed.Column{
		{"b", zed.TypeBstring},
		{"s", zed.TypeString},
	})
	assert.NoError(t, err)

	type Expect struct {
		fmt      OutFmt
		expected string
	}

	cases := []struct {
		val      zed.Value
		expected []Expect
	}{
		//
		// Test bstrings
		//

		// An ascii string
		{
			zed.NewBstring("foo"),
			[]Expect{
				{OutFormatZeek, "foo"},
				{OutFormatZeekAscii, "foo"},
				{OutFormatZNG, "foo"},
			},
		},

		// An null string is represented as -
		{
			zed.Value{zed.TypeBstring, nil},
			[]Expect{
				{OutFormatZeek, "-"},
				{OutFormatZeekAscii, "-"},
				{OutFormatZNG, "-"},
			},
		},

		// A value consisting of just - must be escaped
		{
			zed.NewBstring("-"),
			[]Expect{
				{OutFormatZeek, `\x2d`},
				{OutFormatZeekAscii, `\x2d`},
				{OutFormatZNG, `\x2d`},
			},
		},

		// A longer value containing - is not escaped
		{
			zed.NewBstring("--"),
			[]Expect{
				{OutFormatZeek, "--"},
				{OutFormatZeekAscii, "--"},
				{OutFormatZNG, "--"},
			},
		},

		// Invalid UTF-8 is escaped
		{
			zed.Value{zed.TypeBstring, []byte{0xae, 0x8c, 0x9f, 0xf0}},
			[]Expect{
				{OutFormatZeek, `\xae\x8c\x9f\xf0`},
				{OutFormatZeekAscii, `\xae\x8c\x9f\xf0`},
				{OutFormatZNG, `\xae\x8c\x9f\xf0`},
			},
		},

		// A backslash is escaped
		{
			zed.NewBstring(`\`),
			[]Expect{
				{OutFormatZeek, `\\`},
				{OutFormatZeekAscii, `\\`},
				{OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zed.NewBstring("\n\t"),
			[]Expect{
				{OutFormatZeek, `\x0a\x09`},
				{OutFormatZeekAscii, `\x0a\x09`},
				{OutFormatZNG, `\x0a\x09`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zed.NewBstring("a,b"),
			[]Expect{
				{OutFormatZeek, `a,b`},
				{OutFormatZeekAscii, `a,b`},
				{OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zed.NewBstring("[hello"),
			[]Expect{
				{OutFormatZeek, `[hello`},
				{OutFormatZeekAscii, `[hello`},
				{OutFormatZNG, `\x5bhello`},
			},
		},

		// Square bracket not at the start of a value is not escaped
		{
			zed.NewBstring("hello["),
			[]Expect{
				{OutFormatZeek, `hello[`},
				{OutFormatZeekAscii, `hello[`},
				{OutFormatZNG, `hello[`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zed.NewBstring(";"),
			[]Expect{
				{OutFormatZeek, `;`},
				{OutFormatZeekAscii, `;`},
				{OutFormatZNG, `\x3b`},
			},
		},

		// A non-ascii unicode code point is escaped in zeek-ascii
		// but left intact in other formats.
		{
			zed.NewBstring("🌮"),
			[]Expect{
				{OutFormatZeek, "🌮"},
				{OutFormatZeekAscii, `\xf0\x9f\x8c\xae`},
				{OutFormatZNG, "🌮"},
			},
		},

		//
		// Test string escapes (\u vs \x)
		//

		// A value consisting of just - must be escaped
		{
			zed.NewString("-"),
			[]Expect{
				{OutFormatZeek, `\u002d`},
				{OutFormatZeekAscii, `\u002d`},
				{OutFormatZNG, `\u002d`},
			},
		},

		// A backslash is escaped
		{
			zed.NewString(`\`),
			[]Expect{
				{OutFormatZeek, `\\`},
				{OutFormatZeekAscii, `\\`},
				{OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zed.NewString("\n\t"),
			[]Expect{
				{OutFormatZeek, `\u{a}\u{9}`},
				{OutFormatZeekAscii, `\u{a}\u{9}`},
				{OutFormatZNG, `\u{a}\u{9}`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zed.NewString("a,b"),
			[]Expect{
				{OutFormatZeek, `a,b`},
				{OutFormatZeekAscii, `a,b`},
				{OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zed.NewString("[hello"),
			[]Expect{
				{OutFormatZeek, `[hello`},
				{OutFormatZeekAscii, `[hello`},
				{OutFormatZNG, `\u{5b}hello`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zed.NewString(";"),
			[]Expect{
				{OutFormatZeek, `;`},
				{OutFormatZeekAscii, `;`},
				{OutFormatZNG, `\u{3b}`},
			},
		},

		//
		// Test sets
		//

		// null set
		{
			zed.Value{bstringSetType, nil},
			[]Expect{
				{OutFormatZeek, "-"},
				{OutFormatZeekAscii, "-"},
				{OutFormatZNG, "-"},
			},
		},

		// empty set
		{
			zed.Value{bstringSetType, []byte{}},
			[]Expect{
				{OutFormatZeek, "(empty)"},
				{OutFormatZeekAscii, "(empty)"},
				{OutFormatZNG, "[]"},
			},
		},

		// simple set
		{
			zed.Value{
				bstringSetType,
				makeContainer([]byte("abc"), []byte("xyz")),
			},
			[]Expect{
				{OutFormatZeek, "abc,xyz"},
				{OutFormatZeekAscii, "abc,xyz"},
				{OutFormatZNG, "[abc;xyz;]"},
			},
		},

		// A comma inside a string inside a set is escaped in Zeek.
		{
			zed.Value{bstringSetType, makeContainer([]byte("a,b"))},
			[]Expect{
				{OutFormatZeek, `a\x2cb`},
			},
		},

		// set containing vectors
		{
			zed.Value{
				setOfVectorsType,
				makeContainer(
					makeContainer([]byte("a"), []byte("b")),
					makeContainer([]byte("x"), []byte("y")),
				),
			},
			[]Expect{
				// not representable in zeek
				{OutFormatZNG, `[[a;b;];[x;y;];]`},
			},
		},

		//
		// Test vectors
		//

		// null vector
		{
			zed.Value{bstringVecType, nil},
			[]Expect{
				{OutFormatZeek, "-"},
				{OutFormatZeekAscii, "-"},
				{OutFormatZNG, "-"},
			},
		},

		// empty vector
		{
			zed.Value{bstringVecType, []byte{}},
			[]Expect{
				{OutFormatZeek, "(empty)"},
				{OutFormatZeekAscii, "(empty)"},
				{OutFormatZNG, "[]"},
			},
		},

		// simple vector
		{
			zed.Value{
				bstringVecType,
				makeContainer([]byte("abc"), []byte("xyz")),
			},
			[]Expect{
				{OutFormatZeek, "abc,xyz"},
				{OutFormatZeekAscii, "abc,xyz"},
				{OutFormatZNG, "[abc;xyz;]"},
			},
		},

		// vector containing vectors
		{
			zed.Value{
				vecOfVectorsType,
				makeContainer(
					makeContainer([]byte("a"), []byte("b")),
					makeContainer([]byte("x"), []byte("y")),
				),
			},
			[]Expect{
				// not representable in zeek
				{OutFormatZNG, `[[a;b;];[x;y;];]`},
			},
		},

		// A comma inside a string inside a vector is escaped in Zeek.
		{
			zed.Value{bstringVecType, makeContainer([]byte("a,b"))},
			[]Expect{
				{OutFormatZeek, `a\x2cb`},
			},
		},

		// vector containing null
		{
			zed.Value{
				bstringVecType,
				makeContainer([]byte("-"), nil),
			},
			[]Expect{
				{OutFormatZeek, `\x2d,-`},
				{OutFormatZNG, `[\x2d;-;]`},
			},
		},

		// vector containing empty string
		{
			zed.Value{bstringVecType, makeContainer([]byte{})},
			[]Expect{
				{OutFormatZeek, ""},
				{OutFormatZNG, `[;]`},
			},
		},

		//
		// Test records
		//

		// Simple record
		{
			zed.Value{
				recType,
				makeContainer([]byte("foo"), []byte("bar")),
			},
			[]Expect{
				{OutFormatZNG, `[foo;bar;]`},
			},
		},

		// Record with nils
		{
			zed.Value{recType, makeContainer(nil, nil)},
			[]Expect{
				{OutFormatZNG, `[-;-;]`},
			},
		},
	}
	for _, tc := range cases {
		for _, expect := range tc.expected {
			t.Run(expect.expected, func(t *testing.T) {
				res := FormatValue(tc.val, expect.fmt)
				assert.Equal(t, expect.expected, res)
			})
		}
	}
}
