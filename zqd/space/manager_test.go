package space

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestV1Migration(t *testing.T) {
	t.Run("InvalidCharacters", func(t *testing.T) {
		root, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		defer os.RemoveAll(root)

		testWriteConfig(t, root, config{Name: "name-with &*(&stuff"})

		mgr, err := NewManager(root, zap.NewNop())
		require.NoError(t, err)
		list, err := mgr.List(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, "name_with_stuff", list[0].Name)
	})
	t.Run("DuplicateNames", func(t *testing.T) {
		root, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		defer os.RemoveAll(root)

		testWriteConfig(t, root, config{Name: "testname"})
		testWriteConfig(t, root, config{Name: "testname"})

		mgr, err := NewManager(root, zap.NewNop())
		require.NoError(t, err)
		list, err := mgr.List(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 2)
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		require.Equal(t, "testname", list[0].Name)
		require.Equal(t, "testname_01", list[1].Name)
	})
}

var counter int

func testWriteConfig(t *testing.T, root string, c config) {
	counter++
	spdir := filepath.Join(root, fmt.Sprintf("sp_%d", counter))
	require.NoError(t, os.Mkdir(spdir, 0700))
	require.NoError(t, fs.MarshalJSONFile(c, filepath.Join(spdir, configFile), 0600))
}
