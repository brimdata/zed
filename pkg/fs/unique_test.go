package fs

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueDir(t *testing.T) {
	tdir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tdir)

	err = os.Mkdir(path.Join(tdir, "foo"), 0700)
	require.NoError(t, err)

	_, err = uniqueDir(tdir, "foo", 1)
	require.Error(t, err)
	require.True(t, os.IsExist(err))
}
