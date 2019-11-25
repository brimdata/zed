package ndjson_test

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/skim"
	"github.com/mccanne/zq/pkg/zsio/ndjson"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/stretchr/testify/require"
)

func TestObjectIn(t *testing.T) {
	type testcase struct {
		name, input string
	}
	cases := []testcase{
		{
			name:  "single line",
			input: `{ "string1": "value1", "int1": 1, "double1": 1.2, "bool1": false }`,
		},
		{
			name: "skips empty lines",
			input: `{ "string1": "value1", "int1": 1, "double1": 1.2, "bool1": false }

		{"string1": "value2", "int1": 2, "double1": 2.3, "bool1": true }
		`,
		},
		{
			name: "nested containers",
			input: `{ "obj1": { "obj2": { "double1": 1.1 } } }
		{ "arr1": [ "string1", "string2", "string3" ] }`,
		},
		{
			name:  "null value",
			input: `{ "null1": null }`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runtestcase(t, c.input)
		})
	}
}

func runtestcase(t *testing.T, input string) {
	var out output
	w := ndjson.NewWriter(&out)
	r := ndjson.NewReader(strings.NewReader(input), resolver.NewTable())
	require.NoError(t, zson.Copy(zson.NopFlusher(w), r))
	NDJSONEq(t, input, out.String())
}

type output struct {
	bytes.Buffer
}

func (o *output) Close() error { return nil }

func newSkimmer(in string) *skim.Scanner {
	buffer := make([]byte, ndjson.ReadSize)
	return skim.NewScanner(strings.NewReader(in), buffer, ndjson.MaxLineSize)
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
