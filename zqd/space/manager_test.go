package space

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/iosrc"
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
	s, err := m.Create(context.Background(), api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	id := s.ID()
	versionConfig := struct {
		Version int `json:"version"`
	}{}
	err = fs.UnmarshalJSONFile(filepath.Join(root, string(id), configFile), &versionConfig)
	require.NoError(t, err)

	require.Equal(t, configVersion, versionConfig.Version)
}

func TestV3MigrationNoPcap(t *testing.T) {
	tm := newTestMigration(t)

	id := tm.initSpace(configV2{
		Version:  2,
		Name:     "test",
		DataURI:  iosrc.URI{},
		PcapPath: "",
		Storage: storage.Config{
			Kind: storage.FileStore,
		},
	})
	info := tm.spaceInfo(id)
	conf := tm.spaceConfig(id)

	assert.Equal(t, "test", info.Name)
	assert.Equal(t, "", conf.DataURI.String())
	assert.Equal(t, "", info.PcapPath.String())
	assert.Equal(t, false, info.PcapSupport)
}

func TestV3MigrationPcap(t *testing.T) {
	tm := newTestMigration(t)
	pcapuri := tm.root.AppendPath("test.pcap")

	id := tm.initSpace(configV2{
		Version:  2,
		Name:     "test",
		DataURI:  iosrc.URI{},
		PcapPath: pcapuri.Filepath(),
		Storage: storage.Config{
			Kind: storage.FileStore,
		},
	})
	err := iosrc.WriteFile(context.Background(), pcapuri, nil)
	require.NoError(t, err)
	tm.writeSpaceJSONFile(id, "packets.idx.json", pcap.Index{})

	info := tm.spaceInfo(id)
	conf := tm.spaceConfig(id)

	assert.Equal(t, "test", info.Name)
	assert.Equal(t, "", conf.DataURI.String())
	assert.Equal(t, pcapuri, info.PcapPath)
	assert.Equal(t, true, info.PcapSupport)
}

func TestV2Migration(t *testing.T) {
	tm := newTestMigration(t)
	pcapuri := tm.root.AppendPath("test.pcap")

	id := tm.initSpace(configV1{
		Version:  1,
		Name:     "test",
		DataPath: ".",
		PcapPath: pcapuri.Filepath(),
		Storage: storage.Config{
			Kind: storage.FileStore,
		},
	})
	err := iosrc.WriteFile(context.Background(), pcapuri, nil)
	require.NoError(t, err)
	tm.writeSpaceJSONFile(id, "packets.idx.json", pcap.Index{})

	info := tm.spaceInfo(id)
	assert.Equal(t, "test", info.Name)
	assert.Equal(t, pcapuri, info.PcapPath)
}

func TestV1Migration(t *testing.T) {
	t.Run("InvalidCharacters", func(t *testing.T) {
		tm := newTestMigration(t)

		tm.initSpace(configV1{Name: "name/𝚭𝚴𝚪/stuff"})

		mgr := tm.manager()
		list, err := mgr.List(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, "name_𝚭𝚴𝚪_stuff", list[0].Name)
	})
	t.Run("DuplicateNames", func(t *testing.T) {
		tm := newTestMigration(t)

		tm.initSpace(configV1{Name: "testname"})
		tm.initSpace(configV1{Name: "testname"})

		mgr := tm.manager()
		list, err := mgr.List(context.Background())
		require.NoError(t, err)
		require.Len(t, list, 2)
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		require.Equal(t, "testname", list[0].Name)
		require.Equal(t, "testname_01", list[1].Name)
		require.NotEqual(t, list[0].Name, list[1].Name)
	})
}

type testMigration struct {
	*testing.T
	root    iosrc.URI
	mgr     *Manager
	counter int
}

func newTestMigration(t *testing.T) *testMigration {
	tm := &testMigration{T: t}
	tm.initRoot()
	return tm
}

func (tm *testMigration) initRoot() {
	root, err := ioutil.TempDir("", "")
	require.NoError(tm.T, err)
	tm.Cleanup(func() {
		os.RemoveAll(root)
	})
	u, err := iosrc.ParseURI(root)
	require.NoError(tm, err)
	tm.root = u
}

func (tm *testMigration) manager() *Manager {
	if tm.mgr == nil {
		mgr, err := NewManager(tm.root, zap.NewNop())
		require.NoError(tm.T, err)
		tm.mgr = mgr
	}
	return tm.mgr
}

func (tm *testMigration) spaceInfo(id api.SpaceID) api.SpaceInfo {
	mgr := tm.manager()
	sp, err := mgr.Get(id)
	require.NoError(tm, err)
	info, err := sp.Info(context.Background())
	require.NoError(tm, err)
	return info
}

func (tm *testMigration) spaceConfig(id api.SpaceID) config {
	var c config
	err := fs.UnmarshalJSONFile(filepath.Join(tm.root.Filepath(), id.String(), configFile), &c)
	require.NoError(tm, err)
	assert.Equal(tm, configVersion, c.Version)
	return c
}

func (tm *testMigration) initSpace(c interface{}) api.SpaceID {
	tm.counter++
	id := api.SpaceID(fmt.Sprintf("sp_%d", tm.counter))
	spdir := filepath.Join(tm.root.Filepath(), string(id))
	require.NoError(tm, os.Mkdir(spdir, 0700))
	tm.writeSpaceJSONFile(id, configFile, c)
	return id
}

func (tm *testMigration) writeSpaceJSONFile(id api.SpaceID, filename string, c interface{}) {
	spdir := tm.root.AppendPath(string(id)).Filepath()
	require.NoError(tm, fs.MarshalJSONFile(c, filepath.Join(spdir, filename), 0600))
}
