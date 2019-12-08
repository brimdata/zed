package ndjsonio_test

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/zio/ndjsonio"
	"github.com/mccanne/zq/pkg/zq"
	"github.com/mccanne/zq/pkg/zq/resolver"
	"github.com/stretchr/testify/require"
)

func TestNDJSON(t *testing.T) {
	type testcase struct {
		name, input, output string
	}
	cases := []testcase{
		{
			name:   "single line",
			input:  `{ "string1": "value1", "int1": 1, "double1": 1.2, "bool1": false }`,
			output: "",
		},
		{
			name: "skips empty lines",
			input: `{ "string1": "value1", "int1": 1, "double1": 1.2, "bool1": false }

		{"string1": "value2", "int1": 2, "double1": 2.3, "bool1": true }
		`,
			output: "",
		},
		{
			name: "nested containers",
			input: `{ "obj1": { "obj2": { "double1": 1.1 } } }
		{ "arr1": [ "string1", "string2", "string3" ] }`,
			output: "",
		},
		{
			name:   "null value",
			input:  `{ "null1": null }`,
			output: "",
		},
		{
			name:   "empty array",
			input:  `{ "arr1": [] }`,
			output: "",
		},
		{
			name:   "legacy nested fields",
			input:  `{ "s": "foo", "nest.s": "bar", "nest.n": 5 }`,
			output: `{ "s": "foo", "nest": { "s": "bar", "n": 5 }}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			output := c.output
			if len(output) == 0 {
				output = c.input
			}
			runtestcase(t, c.input, output)
		})
	}
}

func runtestcase(t *testing.T, input, output string) {
	var out bytes.Buffer
	w := ndjsonio.NewWriter(&out)
	r := ndjsonio.NewReader(strings.NewReader(input), resolver.NewTable())
	require.NoError(t, zq.Copy(zq.NopFlusher(w), r))
	NDJSONEq(t, output, out.String())
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
