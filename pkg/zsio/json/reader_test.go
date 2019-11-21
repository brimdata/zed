package json_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	zjson "github.com/mccanne/zq/pkg/zsio/json"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/stretchr/testify/require"
)

func zsonCopy(dst zson.WriteCloser, src zson.Reader) error {
	var err error
	for {
		var rec *zson.Record
		rec, err = src.Read()
		if err != nil || rec == nil {
			break
		}
		err = dst.Write(rec)
		if err != nil {
			break
		}
	}
	dstErr := dst.Close()
	switch {
	case err != nil:
		return err
	case dstErr != nil:
		return dstErr
	default:
		return nil
	}
}

type output struct {
	bytes.Buffer
}

func (o *output) Close() error { return nil }

func testcase(t *testing.T, input string, expected string) {
	var out output
	w := zjson.NewWriter(&out)
	r := zjson.NewReader(strings.NewReader(input), resolver.NewTable())
	require.NoError(t, zsonCopy(w, r))
	require.JSONEq(t, expected, out.String())
}

func TestObjectIn(t *testing.T) {
	input := `{
"string1": "value1",
"int1": 1,
"double1": 1.2,
"bool1": false
}`
	testcase(t, input, fmt.Sprintf("[%s]", input))
}

func TestArrayIn(t *testing.T) {
	input := `[
	{
		"string1": "value1",
		"int1": 1,
		"double1": 1.2,
		"bool1": false
	}, {
		"string1": "value2",
		"int1": 2,
		"double1": 2.3,
		"bool1": true
	}
]`
	testcase(t, input, input)
}
