package zng

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnescape(t *testing.T) {
	cases := []struct {
		unescaped string
		escaped   string
	}{
		{`\`, `\\`},
		{`\\`, `\\\\`},
		{`ascii`, `ascii`},
		{"\a\b\f\n\r\t\v", `\x07\x08\x0c\x0a\x0d\x09\x0b`},
		{"\x00\x19\x20\\\x7e\x7f\xff", "\\x00\\x19\x20\\\\\x7e\\x7f\\xff"},
	}
	for _, c := range cases {
		in, expected := c.escaped, c.unescaped

		actual := Unescape([]byte(in))
		require.Exactly(t, []byte(expected), actual, "case: %#v", c)

		actual = Unescape([]byte("prefix" + in + "suffix"))
		expected = "prefix" + expected + "suffix"
		require.Exactly(t, []byte(expected), actual, "case: %#v", c)
	}
}

func TestUnescapeUTF(t *testing.T) {
	cases := []struct {
		unescaped string
		escaped   string
	}{
		{`\`, `\\`},
		{`\\`, `\\\\`},
		{`ascii`, `ascii`},
		{"\a\b\f\n\r\t\v", `\x07\x08\x0c\x0a\x0d\x09\x0b`},
		{"\x00\x19\x20\\\x7e\x7f\xff", "\\x00\\x19\x20\\\\\x7e\\x7f\\xff"},
		{"\x00😁", `\x00\xf0\x9f\x98\x81`},
	}
	for _, c := range cases {
		in, expected := c.escaped, c.unescaped

		actual := Unescape([]byte(in))
		require.Exactly(t, []byte(expected), actual, "case: %#v", c)

		actual = Unescape([]byte("prefix" + in + "suffix"))
		expected = "prefix" + expected + "suffix"
		require.Exactly(t, []byte(expected), actual, "case: %#v", c)
	}
}
