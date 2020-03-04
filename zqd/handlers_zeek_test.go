// +build zeek

package zqd_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/api"
	"github.com/stretchr/testify/suite"
)

func TestPacketPost(t *testing.T) {
	suite.Run(t, new(PacketPostSuite))
}

type PacketPostSuite struct {
	suite.Suite
	root     string
	space    string
	pcapfile string
}

func (s *PacketPostSuite) TestCount() {
	expected := `
#0:record[count:uint64]
0:[4;]`
	res := execSearch(s.T(), s.root, s.space, "count()")
	s.Equal(test.Trim(expected), res)
}

func (s *PacketPostSuite) TestNoPacketFilter() {
	expected := ""
	res := execSearch(s.T(), s.root, s.space, "_path = packet_filter | count()")
	s.Equal(expected, res)
}

func (s *PacketPostSuite) TestSpaceInfo() {
	u := fmt.Sprintf("http://localhost:9867/space/%s", s.space)
	res := httpSuccess(s.T(), zqd.NewHandler(s.root), "GET", u, nil)
	var info api.SpaceInfo
	err := json.NewDecoder(res).Decode(&info)
	s.NoError(err)
	s.Equal(s.pcapfile, info.PacketPath)
	s.True(info.PacketSupport)
}

func (s *PacketPostSuite) SetupTest() {
	s.space = "test"
	dir, err := ioutil.TempDir("", "PacketPostTest")
	s.NoError(err)
	s.root = dir
	s.pcapfile = filepath.Join(".", "testdata/test.pcap")
	createSpaceWithPcap(s.T(), s.root, s.space, s.pcapfile)
}

func (s *PacketPostSuite) TearDownTest() {
	os.RemoveAll(s.root)
}

func createSpaceWithPcap(t *testing.T, root, spaceName, pcapfile string) {
	createSpace(t, root, spaceName, "")
	req := api.PacketPostRequest{filepath.Join(".", pcapfile)}
	u := fmt.Sprintf("http://localhost:9867/space/%s/packet", spaceName)
	httpSuccess(t, zqd.NewHandler(root), "POST", u, req)
}
