// +build linux darwin
// +build system

package tests

import (
	"path/filepath"
	"testing"

	"github.com/brimsec/zq/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScripts(t *testing.T) {
	t.Parallel()
	path, err := filepath.Abs(zqpath)
	require.NoError(t, err)
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
