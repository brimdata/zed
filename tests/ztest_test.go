package tests

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/brimsec/zq/ztest"
	"github.com/stretchr/testify/require"
)

func TestZTest(t *testing.T) {
	t.Parallel()
	dirs := map[string]bool{}
	re, _ := regexp.Compile("/test/.*\\.yaml")
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".yaml") {
			return err
		}
		if strings.HasPrefix(path, "../tests/") || re.MatchString(path) {
			dirs[filepath.Dir(path)] = true
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
