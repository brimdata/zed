package zngio

import (
	"bytes"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Parallel()
	const input = `
{_path:"a",ts:1970-01-01T00:00:10Z,d:1.}
{_path:"xyz",ts:1970-01-01T00:00:20Z,d:1.5}
`
	expectedHex := `
# define a record with 3 columns
f6 03
# first column name is _path (len 5)
05 5f 70 61 74 68
# first column type is string (16)
10
# second column name is ts (len 2)
02 74 73
# second column type is time (9)
09
# third column name is d (len 1)
01 64
# third column type is float64 (12)
0c
# value using type id 23 (0x17), the record defined above
# total length of this recor is 17 bytes (0x11)
17 11
# first column is a 1-byte value
02
# value of the first column is the string "a"
61
# second column is a 6-byte value
06
# time value is encoded in nanoseconds shifted one bit left
# 2000000000 == 0x04a817c800
00 c8 17 a8 04
# third column is a 9-byte value
09
# 8 bytes of float64 data representing 1.0
00 00 00 00 00 00 f0 3f
# another encoded value using the same record definition as before
17 13
# first column: primitive value of 4 total byte, values xyz
04 78 79 7a
# second column: primitive value of 20 (converted to nanoseconds, encoded <<1)
06 00 90 2f 50 09
# third column, primitive value of 9 total bytes, float64 1.5
09 00 00 00 00 00 00 f8 3f
# end of stream
ff
`
	// Remove all whitespace and comments (from "#" through end of line).
	expectedHex = regexp.MustCompile(`\s|(#[^\n]*\n)`).ReplaceAllString(expectedHex, "")
	expected, err := hex.DecodeString(expectedHex)
	require.NoError(t, err)

	zr := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	var buf bytes.Buffer
	zw := NewWriter(zio.NopCloser(&buf), WriterOpts{})
	require.NoError(t, zio.Copy(zw, zr))
	require.NoError(t, zw.Close())
	assert.Equal(t, expected, buf.Bytes())
}
