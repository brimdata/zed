package tests

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoAbsPathLocation(t *testing.T) {
	t.Parallel()
	dir, err := RepoAbsPath()
	require.Equal(t, nil, err)
	f := filepath.Join(dir, "tests", "path.go")
	assert.FileExists(t, f, "%s not in expected repo path pkg/test/path.go", f)
}
