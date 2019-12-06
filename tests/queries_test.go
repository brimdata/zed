// +build system

package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/mccanne/zq/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternal(t *testing.T) {
	t.Parallel()
	for _, d := range internals {
		t.Run(d.Name, func(t *testing.T) {
			results, err := d.Run()
			require.NoError(t, err)
			assert.Exactly(t, d.Expected, results, "Wrong query results")
		})
	}
}

func TestCommands(t *testing.T) {
	t.Parallel()
	path := findPath()
	seen := make(map[string]struct{})
	for _, cmd := range commands {
		name := cmd.Name
		if _, ok := seen[name]; ok {
			t.Logf("test %s: skipping extra (unique test names are required)", name)
		}
		seen[name] = struct{}{}
		t.Run(name, func(t *testing.T) {
			results, err := cmd.Run(path)
			require.NoError(t, err)
			assert.Exactly(t, cmd.Expected, results, "Wrong command results")
		})
	}
}

func findPath() string {
	for _, s := range os.Args {
		if strings.HasPrefix(s, "PATH=") {
			return s[5:]
		}
	}
	return ""
}

func TestScripts(t *testing.T) {
	t.Parallel()
	path := findPath()
	for _, script := range scripts {
		t.Run(script.Name, func(t *testing.T) {
			var fail bool
			shell := test.NewShellTest(script)
			_, _, err := shell.Run(RootDir, path)
			if err != nil {
				fail = true
			}
			require.NoError(t, err)
			for _, file := range script.Expected {
				actual, err := shell.Read(file.Name)
				if err != nil {
					fail = true
				}
				require.NoError(t, err)
				if !assert.Exactly(t, file.Data, actual, "Wrong shell script results") {
					fail = true
				}
			}
			if !fail {
				// Remove the testdir on success.  If test fails,  we
				// leave it behind in testroot for debugging.  These
				// failed test directories have to be manually removed.
				shell.Cleanup()
			}
		})
	}
}
