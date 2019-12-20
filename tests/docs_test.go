// +build system
// +build docs

package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mccanne/zq/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var doctests = []struct {
	name string
	cmd  string

	expected string
}{
	{
		"value-match",
		"zq -f table '10.150.0.85' ../../zq-sample-data/zeek-default/*.log",
		"value-match.table",
	},
	{
		"quoted-word",
		`zq -f table '"O=Internet Widgits"' ../../zq-sample-data/zeek-default/*.log`,
		"quoted-word.table",
	},
}

func execs() ([]test.Exec, error) {
	var tests []test.Exec

	for _, tt := range doctests {
		exp, err := os.Open(filepath.Join("testdata", "zqldocs", tt.expected))
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		_, err = buf.ReadFrom(exp)
		if err != nil {
			return nil, err
		}
		s := buf.String()
		tests = append(tests, test.Exec{
			Name:     tt.name,
			Command:  tt.cmd,
			Expected: s,
		})
	}
	return tests, nil
}

func TestDocExamples(t *testing.T) {
	t.Parallel()
	path := findPath()
	docTests, err := execs()
	require.NoError(t, err)
	for _, cmd := range docTests {
		name := cmd.Name
		t.Run(name, func(t *testing.T) {
			results, err := cmd.Run(path)
			require.NoError(t, err)
			assert.Exactly(t, cmd.Expected, results, "Wrong results")
		})
	}
}
