package zq

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/brimsec/zq/ztest"
	"github.com/stretchr/testify/require"
)

func TestZq(t *testing.T) {
	t.Parallel()
	dirs := map[string]struct{}{}
	re, _ := regexp.Compile(".*ztests/.*\\.yaml$")
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".yaml") && (re.MatchString(path) || strings.HasPrefix(path, "tests/")) {
			dirs[filepath.Dir(path)] = struct{}{}
		}
		return err
	})
	require.NoError(t, err)
	for d := range dirs {
		d := d
		t.Run(d, func(t *testing.T) {
			t.Parallel()
			ztest.Run(t, d)
		})
	}
}
