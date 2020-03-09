// +build zeek

package zqd_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	zeekpath = os.Getenv("ZEEK")
)

func TestPacketPostSuccess(t *testing.T) {
	p := packetPost(t, zeekpath, "./testdata/valid.pcap", 202)
	defer p.cleanup()
	t.Run("DataReverseSorted", func(t *testing.T) {
		expected := `
#0:record[ts:time]
0:[1501770880.988247;]
0:[1501770877.501001;]
0:[1501770877.471635;]
0:[1501770877.471635;]`
		res := execSearch(t, p.core, p.space, "cut ts")
		assert.Equal(t, test.Trim(expected), res)
	})
	t.Run("SpaceInfo", func(t *testing.T) {
		u := fmt.Sprintf("http://localhost:9867/space/%s", p.space)
		var info api.SpaceInfo
		httpJSONSuccess(t, zqd.NewHandler(p.core), "GET", u, nil, &info)
		assert.Equal(t, p.space, info.Name)
		assert.Equal(t, nano.Unix(1501770877, 471635000), *info.MinTime)
		assert.Equal(t, nano.Unix(1501770880, 988247000), *info.MaxTime)
		// XXX Must use InDelta here because zeek's randomly generate uids can
		// vary by 1 characater in size. Should probably be tested with the
		// same seed set in zeek.
		assert.InDelta(t, 1437, info.Size, 2)
		assert.True(t, info.PacketSupport)
		assert.Equal(t, p.pcapfile, info.PacketPath)
	})
	t.Run("PacketIndexExists", func(t *testing.T) {
		require.FileExists(t, filepath.Join(p.core.Root, p.space, packet.IndexFile))
	})
	t.Run("ResponseMessages", func(t *testing.T) {
		info, err := os.Stat(p.pcapfile)
		require.NoError(t, err)
		status := p.payloads[1].(*api.PacketPostStatus)
		assert.Equal(t, status.Type, "PacketPostStatus")
		assert.Equal(t, status.PacketSize, info.Size())
		assert.Equal(t, status.PacketReadSize, info.Size())
	})
}

func TestPacketPostInvalidPcap(t *testing.T) {
	p := packetPost(t, zeekpath, "./testdata/invalid.pcap", 500)
	defer p.cleanup()
	t.Run("ErrorMessage", func(t *testing.T) {
		// XXX Better error message here.
		require.Regexp(t, "^Unknown magic*", string(p.body))
	})
}

func TestPacketPostZeekFailImmediate(t *testing.T) {
	exec := abspath(t, filepath.Join("testdata", "zeekstartfail.sh"))
	p := packetPost(t, exec, "./testdata/valid.pcap", 202)
	defer p.cleanup()
	t.Run("TaskEndError", func(t *testing.T) {
		expected := &api.TaskEnd{
			Type: "TaskEnd",
			// XXX This is dependent on execution order. TaskID is global when
			// should be attached to instance of core.
			TaskID: 2,
			Error: &api.Error{
				Type: "Error",
				// XXX This is not an informative failure message. Will fix in
				// followup pr.
				Message: "exit status 2",
			},
		}
		require.Contains(t, p.payloads, expected)
	})
}

func TestPacketPostZeekFailAfterWrite(t *testing.T) {
	exec := abspath(t, filepath.Join("testdata", "zeekwritefail.sh"))
	p := packetPost(t, exec, "./testdata/valid.pcap", 202)
	defer p.cleanup()
	t.Run("TaskEndError", func(t *testing.T) {
		expected := &api.TaskEnd{
			Type: "TaskEnd",
			// XXX This is dependent on execution order. TaskID is global when
			// should be attached to instance of core.
			TaskID: 3,
			Error: &api.Error{
				Type: "Error",
				// XXX This is not an informative failure message. Will fix in
				// followup pr.
				Message: "exit status 1",
			},
		}
		require.Contains(t, p.payloads, expected)
	})
	t.Run("EmptySpaceInfo", func(t *testing.T) {
		u := fmt.Sprintf("http://localhost:9867/space/%s", p.space)
		var info api.SpaceInfo
		httpJSONSuccess(t, zqd.NewHandler(p.core), "GET", u, nil, &info)
		expected := api.SpaceInfo{
			Name: p.space,
		}
		require.Equal(t, expected, info)
	})
}

func packetPost(t *testing.T, zeekExec, pcapfile string, expectedStatus int) packetPostResult {
	if zeekExec == "" {
		zeekExec = zeekpath
	}
	if pcapfile == "" {
		pcapfile = "./testdata/test.pcap"
	}
	dir, err := ioutil.TempDir("", "PacketPostTest")
	require.NoError(t, err)
	res := packetPostResult{
		core:     &zqd.Core{Root: dir, ZeekExec: zeekExec},
		space:    "test",
		pcapfile: pcapfile,
	}
	res.postPcap(t, pcapfile)
	require.Equalf(t, expectedStatus, res.statusCode, "unexpected status code: %s", string(res.body))
	return res
}

type packetPostResult struct {
	core       *zqd.Core
	space      string
	statusCode int
	pcapfile   string
	body       []byte
	payloads   []interface{}
}

func (r *packetPostResult) postPcap(t *testing.T, file string) {
	createSpace(t, r.core, r.space, "")
	u := fmt.Sprintf("http://localhost:9867/space/%s/packet", r.space)
	res := httpRequest(t, zqd.NewHandler(r.core), "POST", u, api.PacketPostRequest{r.pcapfile})
	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	r.body, r.statusCode = body, res.StatusCode
	if r.statusCode == 202 {
		r.readPayloads(t)
	}
}

func (r *packetPostResult) readPayloads(t *testing.T) {
	scanner := api.NewJSONPipeScanner(bytes.NewReader(r.body))
	_, cancel := context.WithCancel(context.Background())
	stream := api.NewStream(scanner, cancel)
	for {
		i, err := stream.Next()
		require.NoError(t, err)
		if i == nil {
			break
		}
		r.payloads = append(r.payloads, i)
	}
}

func (r *packetPostResult) cleanup() {
	os.RemoveAll(r.core.Root)
}

func abspath(t *testing.T, path string) string {
	p, err := filepath.Abs(path)
	require.NoError(t, err)
	return p
}
