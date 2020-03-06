// +build zeek

package zqd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/packet"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestPacketPost(t *testing.T) {
	suite.Run(t, new(PacketPostSuite))
}

type PacketPostSuite struct {
	suite.Suite
	core     *zqd.Core
	space    string
	pcapfile string
	payloads []interface{}
}

func (s *PacketPostSuite) TestCount() {
	expected := `
#0:record[count:uint64]
0:[4;]`
	res := execSearch(s.T(), s.core, s.space, "count()")
	s.Equal(test.Trim(expected), res)
}

func (s *PacketPostSuite) TestNoPacketFilter() {
	expected := ""
	res := execSearch(s.T(), s.core, s.space, "_path = packet_filter | count()")
	s.Equal(expected, res)
}

func (s *PacketPostSuite) TestSpaceInfo() {
	u := fmt.Sprintf("http://localhost:9867/space/%s", s.space)
	res := httpSuccess(s.T(), zqd.NewHandler(s.core), "GET", u, nil)
	var info api.SpaceInfo
	err := json.NewDecoder(res).Decode(&info)
	s.NoError(err)
	s.Equal(s.pcapfile, info.PacketPath)
	s.True(info.PacketSupport)
}

func (s *PacketPostSuite) TestWritesIndexFile() {
	stat, err := os.Stat(filepath.Join(s.core.Root, s.space, packet.IndexFile))
	s.NoError(err)
	s.NotNil(stat)
}

func (s *PacketPostSuite) TestStatus() {
	info, err := os.Stat(s.pcapfile)
	s.NoError(err)
	s.Len(s.payloads, 3)
	status := s.payloads[1].(*api.PacketPostStatus)
	s.Equal(status.Type, "PacketPostStatus")
	s.Equal(status.PacketSize, info.Size())
	s.Equal(status.PacketReadSize, info.Size())
}

func (s *PacketPostSuite) SetupTest() {
	s.space = "test"
	dir, err := ioutil.TempDir("", "PacketPostTest")
	s.NoError(err)
	s.core = &zqd.Core{Root: dir}
	s.pcapfile = filepath.Join(".", "testdata/test.pcap")
	s.payloads = createSpaceWithPcap(s.T(), s.core, s.space, s.pcapfile)
}

func (s *PacketPostSuite) TearDownTest() {
	os.RemoveAll(s.core.Root)
}

func createSpaceWithPcap(t *testing.T, core *zqd.Core, spaceName, pcapfile string) []interface{} {
	createSpace(t, core, spaceName, "")
	req := api.PacketPostRequest{filepath.Join(".", pcapfile)}
	u := fmt.Sprintf("http://localhost:9867/space/%s/packet", spaceName)
	body := httpSuccess(t, zqd.NewHandler(core), "POST", u, req)
	scanner := api.NewJSONPipeScanner(body)
	_, cancel := context.WithCancel(context.Background())
	stream := api.NewStream(scanner, cancel)
	var taskEnd api.TaskEnd
	var payloads []interface{}
	for {
		i, err := stream.Next()
		require.NoError(t, err)
		if i == nil {
			break
		}
		payloads = append(payloads, i)
		if end, ok := i.(api.TaskEnd); ok {
			taskEnd = end
		}
	}
	require.Nil(t, taskEnd.Error)
	return payloads
}
