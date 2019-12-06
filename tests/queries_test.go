// +build system

package tests

import (
	"testing"

	"github.com/mccanne/zq/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func systest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping system test during unit test run")
	}
}

func TestInternal(t *testing.T) {
	systest(t)
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
	systest(t)
	for _, cmd := range commands {
		t.Run(cmd.Name, func(t *testing.T) {
			results, err := cmd.Run()
			require.NoError(t, err)
			assert.Exactly(t, cmd.Expected, results, "Wrong command results")
		})
	}
}

func TestScripts(t *testing.T) {
	systest(t)
	for _, script := range scripts {
		t.Run(script.Name, func(t *testing.T) {
			var fail bool
			shell := test.NewShellTest(script)
			_, _, err := shell.Run(RootDir)
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
