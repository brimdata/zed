package ndjsonio

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/require"
)

// can't use zio.NopCloser since it creates an import cycle
type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// NopCloser returns a WriteCloser with a no-op Close method wrapping
// the provided Writer w.
func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}

func TestNDJSONWriter(t *testing.T) {
	type testcase struct {
		name, input, output string
	}
	cases := []testcase{
		{
			name:   "null containers",
			input:  `{dns:["google.com"],uri:null (0=([string])),email:null (1=(|[string]|)),ip:null (2=([ip]))}`,
			output: `{"dns":["google.com"],"email":null,"ip":null,"uri":null}`,
		},
		{
			name:   "nested nulls",
			input:  `{san:{dns:["google.com"],uri:null (0=([string])),email:null (1=(|[string]|)),ip:null (2=([ip]))}}`,
			output: `{"san":{"dns":["google.com"],"email":null,"ip":null,"uri":null}}`,
		},
		{
			name: "empty containers",
			input: `{dns:["google.com"],uri:[] (0=([string])),email:|[]| (1=(|[string]|)),ip:null (2=([ip]))}
`,
			output: `{"dns":["google.com"],"uri":[], "email":[],"ip":null}`,
		},
		{
			name: "nested empties",
			input: `{san:{dns:["google.com"],uri:[] (0=([string])),email:|[]| (1=(|[string]|)),ip:null (2=([ip]))}}
`,
			output: `{"san":{"dns":["google.com"],"uri":[], "email":[],"ip":null}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out bytes.Buffer
			w := NewWriter(NopCloser(&out))
			r := zson.NewReader(strings.NewReader(c.input), zson.NewContext())
			require.NoError(t, zbuf.Copy(w, r))
			NDJSONEq(t, c.output, out.String())
		})
	}
}

func getLines(in string) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(in))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		lines = append(lines, string(line))
	}
	return lines, scanner.Err()
}

func NDJSONEq(t *testing.T, expected string, actual string) {
	expectedLines, err := getLines(expected)
	require.NoError(t, err)
	actualLines, err := getLines(actual)
	require.NoError(t, err)
	require.Len(t, expectedLines, len(actualLines))
	for i := range actualLines {
		require.JSONEq(t, expectedLines[i], actualLines[i])
	}
}
