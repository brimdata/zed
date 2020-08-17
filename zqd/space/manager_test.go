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
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConfigCurrentVersion(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)
	u, err := iosrc.ParseURI(root)
	require.NoError(t, err)
	m, err := NewManager(u, zap.NewNop())
	require.NoError(t, err)
	s, err := m.Create(api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	id := s.ID()
	versionConfig := struct {
		Version int `json:"version"`
	}{}
	err = fs.UnmarshalJSONFile(filepath.Join(root, string(id), configFile), &versionConfig)
	require.NoError(t, err)

	require.Equal(t, configVersion, versionConfig.Version)
}

func TestV2Migration(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)
	u, err := iosrc.ParseURI(root)
	require.NoError(t, err)

	id := testWriteConfig(t, root, configV1{
		Version:  1,
		Name:     "test",
		DataPath: ".",
		PcapPath: "/tmp/dir",
		Storage: storage.Config{
			Kind: storage.FileStore,
		},
	})

	_, err = NewManager(u, zap.NewNop())
	require.NoError(t, err)
	var c config
	err = fs.UnmarshalJSONFile(filepath.Join(root, id.String(), configFile), &c)
	require.NoError(t, err)

	assert.Equal(t, "test", c.Name)
	assert.Equal(t, "", c.DataURI.String())
	assert.Equal(t, "/tmp/dir", c.PcapPath)
	assert.Equal(t, 2, c.Version)
}

func TestV1Migration(t *testing.T) {
	t.Run("InvalidCharacters", func(t *testing.T) {
		root, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		defer os.RemoveAll(root)
		u, err := iosrc.ParseURI(root)
		require.NoError(t, err)

		testWriteConfig(t, root, configV1{Name: "name/𝚭𝚴𝚪/stuff"})

		mgr, err := NewManager(u, zap.NewNop())
		require.NoError(t, err)
		list, err := mgr.List(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, "name_𝚭𝚴𝚪_stuff", list[0].Name)
	})
	t.Run("DuplicateNames", func(t *testing.T) {
		root, err := ioutil.TempDir("", "")
		require.NoError(t, err)
		defer os.RemoveAll(root)
		u, err := iosrc.ParseURI(root)
		require.NoError(t, err)

		testWriteConfig(t, root, configV1{Name: "testname"})
		testWriteConfig(t, root, configV1{Name: "testname"})

		mgr, err := NewManager(u, zap.NewNop())
		require.NoError(t, err)
		list, err := mgr.List(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 2)
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		require.Equal(t, "testname", list[0].Name)
		require.Regexp(t, "testname_01", list[1].Name)
		require.NotEqual(t, list[0].Name, list[1].Name)
	})
}

var counter int

func testWriteConfig(t *testing.T, root string, c interface{}) api.SpaceID {
	counter++
	id := fmt.Sprintf("sp_%d", counter)
	spdir := filepath.Join(root, id)
	require.NoError(t, os.Mkdir(spdir, 0700))
	require.NoError(t, fs.MarshalJSONFile(c, filepath.Join(spdir, configFile), 0600))
	return api.SpaceID(id)
}
