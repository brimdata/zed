package tzngio_test

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/tzngio"
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
		fmt      tzngio.OutFmt
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
				{tzngio.OutFormatZeek, "foo"},
				{tzngio.OutFormatZeekAscii, "foo"},
				{tzngio.OutFormatZNG, "foo"},
			},
		},

		// An unset string is represented as -
		{
			zed.Value{zed.TypeBstring, nil},
			[]Expect{
				{tzngio.OutFormatZeek, "-"},
				{tzngio.OutFormatZeekAscii, "-"},
				{tzngio.OutFormatZNG, "-"},
			},
		},

		// A value consisting of just - must be escaped
		{
			zed.NewBstring("-"),
			[]Expect{
				{tzngio.OutFormatZeek, `\x2d`},
				{tzngio.OutFormatZeekAscii, `\x2d`},
				{tzngio.OutFormatZNG, `\x2d`},
			},
		},

		// A longer value containing - is not escaped
		{
			zed.NewBstring("--"),
			[]Expect{
				{tzngio.OutFormatZeek, "--"},
				{tzngio.OutFormatZeekAscii, "--"},
				{tzngio.OutFormatZNG, "--"},
			},
		},

		// Invalid UTF-8 is escaped
		{
			zed.Value{zed.TypeBstring, []byte{0xae, 0x8c, 0x9f, 0xf0}},
			[]Expect{
				{tzngio.OutFormatZeek, `\xae\x8c\x9f\xf0`},
				{tzngio.OutFormatZeekAscii, `\xae\x8c\x9f\xf0`},
				{tzngio.OutFormatZNG, `\xae\x8c\x9f\xf0`},
			},
		},

		// A backslash is escaped
		{
			zed.NewBstring(`\`),
			[]Expect{
				{tzngio.OutFormatZeek, `\\`},
				{tzngio.OutFormatZeekAscii, `\\`},
				{tzngio.OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zed.NewBstring("\n\t"),
			[]Expect{
				{tzngio.OutFormatZeek, `\x0a\x09`},
				{tzngio.OutFormatZeekAscii, `\x0a\x09`},
				{tzngio.OutFormatZNG, `\x0a\x09`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zed.NewBstring("a,b"),
			[]Expect{
				{tzngio.OutFormatZeek, `a,b`},
				{tzngio.OutFormatZeekAscii, `a,b`},
				{tzngio.OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zed.NewBstring("[hello"),
			[]Expect{
				{tzngio.OutFormatZeek, `[hello`},
				{tzngio.OutFormatZeekAscii, `[hello`},
				{tzngio.OutFormatZNG, `\x5bhello`},
			},
		},

		// Square bracket not at the start of a value is not escaped
		{
			zed.NewBstring("hello["),
			[]Expect{
				{tzngio.OutFormatZeek, `hello[`},
				{tzngio.OutFormatZeekAscii, `hello[`},
				{tzngio.OutFormatZNG, `hello[`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zed.NewBstring(";"),
			[]Expect{
				{tzngio.OutFormatZeek, `;`},
				{tzngio.OutFormatZeekAscii, `;`},
				{tzngio.OutFormatZNG, `\x3b`},
			},
		},

		// A non-ascii unicode code point is escaped in zeek-ascii
		// but left intact in other formats.
		{
			zed.NewBstring("🌮"),
			[]Expect{
				{tzngio.OutFormatZeek, "🌮"},
				{tzngio.OutFormatZeekAscii, `\xf0\x9f\x8c\xae`},
				{tzngio.OutFormatZNG, "🌮"},
			},
		},

		//
		// Test string escapes (\u vs \x)
		//

		// A value consisting of just - must be escaped
		{
			zed.NewString("-"),
			[]Expect{
				{tzngio.OutFormatZeek, `\u002d`},
				{tzngio.OutFormatZeekAscii, `\u002d`},
				{tzngio.OutFormatZNG, `\u002d`},
			},
		},

		// A backslash is escaped
		{
			zed.NewString(`\`),
			[]Expect{
				{tzngio.OutFormatZeek, `\\`},
				{tzngio.OutFormatZeekAscii, `\\`},
				{tzngio.OutFormatZNG, `\\`},
			},
		},

		// Newlines and tabs are escaped
		{
			zed.NewString("\n\t"),
			[]Expect{
				{tzngio.OutFormatZeek, `\u{a}\u{9}`},
				{tzngio.OutFormatZeekAscii, `\u{a}\u{9}`},
				{tzngio.OutFormatZNG, `\u{a}\u{9}`},
			},
		},

		// Commas not inside a container are not escaped
		{
			zed.NewString("a,b"),
			[]Expect{
				{tzngio.OutFormatZeek, `a,b`},
				{tzngio.OutFormatZeekAscii, `a,b`},
				{tzngio.OutFormatZNG, `a,b`},
			},
		},

		// Square bracket at the start of a value is escaped in ZNG
		{
			zed.NewString("[hello"),
			[]Expect{
				{tzngio.OutFormatZeek, `[hello`},
				{tzngio.OutFormatZeekAscii, `[hello`},
				{tzngio.OutFormatZNG, `\u{5b}hello`},
			},
		},

		// Semicolon is escaped in ZNG
		{
			zed.NewString(";"),
			[]Expect{
				{tzngio.OutFormatZeek, `;`},
				{tzngio.OutFormatZeekAscii, `;`},
				{tzngio.OutFormatZNG, `\u{3b}`},
			},
		},

		//
		// Test sets
		//

		// unset set
		{
			zed.Value{bstringSetType, nil},
			[]Expect{
				{tzngio.OutFormatZeek, "-"},
				{tzngio.OutFormatZeekAscii, "-"},
				{tzngio.OutFormatZNG, "-"},
			},
		},

		// empty set
		{
			zed.Value{bstringSetType, []byte{}},
			[]Expect{
				{tzngio.OutFormatZeek, "(empty)"},
				{tzngio.OutFormatZeekAscii, "(empty)"},
				{tzngio.OutFormatZNG, "[]"},
			},
		},

		// simple set
		{
			zed.Value{
				bstringSetType,
				makeContainer([]byte("abc"), []byte("xyz")),
			},
			[]Expect{
				{tzngio.OutFormatZeek, "abc,xyz"},
				{tzngio.OutFormatZeekAscii, "abc,xyz"},
				{tzngio.OutFormatZNG, "[abc;xyz;]"},
			},
		},

		// A comma inside a string inside a set is escaped in Zeek.
		{
			zed.Value{bstringSetType, makeContainer([]byte("a,b"))},
			[]Expect{
				{tzngio.OutFormatZeek, `a\x2cb`},
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
				{tzngio.OutFormatZNG, `[[a;b;];[x;y;];]`},
			},
		},

		//
		// Test vectors
		//

		// unset vector
		{
			zed.Value{bstringVecType, nil},
			[]Expect{
				{tzngio.OutFormatZeek, "-"},
				{tzngio.OutFormatZeekAscii, "-"},
				{tzngio.OutFormatZNG, "-"},
			},
		},

		// empty vector
		{
			zed.Value{bstringVecType, []byte{}},
			[]Expect{
				{tzngio.OutFormatZeek, "(empty)"},
				{tzngio.OutFormatZeekAscii, "(empty)"},
				{tzngio.OutFormatZNG, "[]"},
			},
		},

		// simple vector
		{
			zed.Value{
				bstringVecType,
				makeContainer([]byte("abc"), []byte("xyz")),
			},
			[]Expect{
				{tzngio.OutFormatZeek, "abc,xyz"},
				{tzngio.OutFormatZeekAscii, "abc,xyz"},
				{tzngio.OutFormatZNG, "[abc;xyz;]"},
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
				{tzngio.OutFormatZNG, `[[a;b;];[x;y;];]`},
			},
		},

		// A comma inside a string inside a vector is escaped in Zeek.
		{
			zed.Value{bstringVecType, makeContainer([]byte("a,b"))},
			[]Expect{
				{tzngio.OutFormatZeek, `a\x2cb`},
			},
		},

		// vector containing unset
		{
			zed.Value{
				bstringVecType,
				makeContainer([]byte("-"), nil),
			},
			[]Expect{
				{tzngio.OutFormatZeek, `\x2d,-`},
				{tzngio.OutFormatZNG, `[\x2d;-;]`},
			},
		},

		// vector containing empty string
		{
			zed.Value{bstringVecType, makeContainer([]byte{})},
			[]Expect{
				{tzngio.OutFormatZeek, ""},
				{tzngio.OutFormatZNG, `[;]`},
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
				{tzngio.OutFormatZNG, `[foo;bar;]`},
			},
		},

		// Record with nils
		{
			zed.Value{recType, makeContainer(nil, nil)},
			[]Expect{
				{tzngio.OutFormatZNG, `[-;-;]`},
			},
		},
	}
	for _, tc := range cases {
		for _, expect := range tc.expected {
			t.Run(expect.expected, func(t *testing.T) {
				res := tzngio.FormatValue(tc.val, expect.fmt)
				assert.Equal(t, expect.expected, res)
			})
		}
	}
}
