// +build system

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimsec/zq/ztest"
	"github.com/stretchr/testify/require"
)

func TestZTest(t *testing.T) {
	t.Parallel()
	dirs := map[string]bool{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".yaml") {
			dirs[filepath.Dir(path)] = true
		}
		return err
	})
	bindir, _ := filepath.Abs("../dist")
	require.NoError(t, err)
	for d := range dirs {
		d := d
		t.Run(d, func(t *testing.T) {
			t.Parallel()
			ztest.Run(t, d, bindir)
		})
	}
}
