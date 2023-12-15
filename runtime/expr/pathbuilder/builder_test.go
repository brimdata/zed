package pathbuilder

import (
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr/dynfield"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/require"
)

func parsePath(zctx *zed.Context, ss ...string) dynfield.Path {
	var path dynfield.Path
	for _, s := range ss {
		path = append(path, *zson.MustParseValue(zctx, s))
	}
	return path
}

type testCase struct {
	describe string
	base     string
	paths    [][]string
	values   []string
	expected string
}

func runTestCase(t *testing.T, c testCase) {
	zctx := zed.NewContext()
	var baseTyp zed.Type
	var baseBytes []byte
	if c.base != "" {
		base := zson.MustParseValue(zctx, c.base)
		baseTyp, baseBytes = base.Type, base.Bytes()
	}
	var paths []dynfield.Path
	for _, ss := range c.paths {
		paths = append(paths, parsePath(zctx, ss...))
	}
	var values []zed.Value
	for _, s := range c.values {
		values = append(values, *zson.MustParseValue(zctx, s))
	}
	step, err := New(baseTyp, paths, values)
	require.NoError(t, err)
	var b zcode.Builder
	typ, err := step.Build(zctx, &b, baseBytes, values)
	require.NoError(t, err)
	val := zed.NewValue(typ, b.Bytes())
	require.Equal(t, c.expected, zson.FormatValue(val))
}

func TestIt(t *testing.T) {
	runTestCase(t, testCase{
		base: `{"a": 1, "b": 2}`,
		paths: [][]string{
			{`"c"`, `"a"`, `"a"`},
			{`"c"`, `"b"`},
			{`"c"`, `"c"`},
		},
		values: []string{
			`45`,
			`"string"`,
			"127.0.0.1",
		},
		expected: `{a:1,b:2,c:{a:{a:45},b:"string",c:127.0.0.1}}`,
	})
	runTestCase(t, testCase{
		base: `{"a": [1,{foo:"bar"}]}`,
		paths: [][]string{
			{`"a"`, `0`},
			{`"a"`, `1`, `"foo"`},
		},
		values: []string{
			`"hi"`,
			`"baz"`,
		},
		expected: `{a:["hi",{foo:"baz"}]}`,
	})
	runTestCase(t, testCase{
		describe: "create from empty base",
		paths: [][]string{
			{`"a"`},
			{`"b"`},
		},
		values: []string{
			`"foo"`,
			`"bar"`,
		},
		expected: `{a:"foo",b:"bar"}`,
	})
	runTestCase(t, testCase{
		describe: "assign to base level array",
		base:     `["a", "b", "c"]`,
		paths: [][]string{
			{`0`},
			{`1`},
			{`2`},
		},
		values: []string{
			`"foo"`,
			`"bar"`,
			`"baz"`,
		},
		expected: `["foo","bar","baz"]`,
	})
}
