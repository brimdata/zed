// +build system

package tests

import (
	"errors"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/brimsec/zq/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const zqpath = "../dist"

func TestInternal(t *testing.T) {
	t.Parallel()
	for _, d := range internals {
		t.Run(d.Name, func(t *testing.T) {
			results, err := d.Run()
			assert.True(t, errors.Is(err, d.ExpectedErr), "expected %v error, got %v", d.ExpectedErr, err)
			assert.Exactly(t, d.Expected, results, "Wrong query results")
		})
	}
}

func TestCommands(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{})
	for _, cmd := range commands {
		name := cmd.Name
		if _, ok := seen[name]; ok {
			t.Logf("test %s: skipping extra (unique test names are required)", name)
		}
		seen[name] = struct{}{}
		t.Run(name, func(t *testing.T) {
			results, err := cmd.Run(zqpath)
			require.NoError(t, err)
			assert.Exactly(t, cmd.Expected, results, "Wrong command results")
		})
	}
}

func TestScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	t.Parallel()
	path, err := filepath.Abs(zqpath)
	require.NoError(t, err)
	for _, script := range scripts {
		t.Run(script.Name, func(t *testing.T) {
			shell := test.NewShellTest(script)
			_, stderr, err := shell.Run(RootDir, path)
			if script.ExpectedStderrRE != "" {
				assert.Regexp(t, regexp.MustCompile(script.ExpectedStderrRE), stderr)
			} else {
				assert.NoError(t, err)
			}
			for _, file := range script.Expected {
				actual, err := shell.Read(file.Name)
				assert.NoError(t, err)
				assert.Exactly(t, file.Data, actual, "Wrong shell script results")
			}
			if !t.Failed() {
				// Remove the testdir on success.  If test fails,  we
				// leave it behind in testroot for debugging.  These
				// failed test directories have to be manually removed.
				shell.Cleanup()
			}
		})
	}
}
