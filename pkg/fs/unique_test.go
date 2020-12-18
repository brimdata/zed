package fs

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueDir(t *testing.T) {
	tdir := t.TempDir()
	err := os.Mkdir(path.Join(tdir, "foo"), 0700)
	require.NoError(t, err)

	_, err = uniqueDir(tdir, "foo", 1)
	require.Error(t, err)
	require.True(t, os.IsExist(err))
}
