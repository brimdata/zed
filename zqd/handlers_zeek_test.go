// +build zeek

package zqd_test

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqd/zeek"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

var testZeekLogs = []string{
	"./testdata/conn.log",
	"./testdata/capture_loss.log",
	"./testdata/http.log",
	"./testdata/stats.log",
}

func TestPacketPostSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}
	ln, err := zeek.LauncherFromPath(os.Getenv("ZEEK"))
	require.NoError(t, err)
	p := packetPost(t, "./testdata/valid.pcap", ln)
	defer p.cleanup()
	t.Run("DataReverseSorted", func(t *testing.T) {
		expected := `
#0:record[ts:time]
0:[1501770880.988247;]
0:[1501770877.501001;]
0:[1501770877.471635;]
0:[1501770877.471635;]`
		res := zngSearch(t, p.client, p.space, "cut ts")
		assert.Equal(t, test.Trim(expected), res)
	})
	t.Run("SpaceInfo", func(t *testing.T) {
		info, err := p.client.SpaceInfo(context.Background(), p.space)
		assert.NoError(t, err)
		assert.Equal(t, p.space, info.Name)
		assert.Equal(t, nano.NewSpanTs(nano.Unix(1501770877, 471635000), nano.Unix(1501770880, 988247000)), *info.Span)
		// Must use InDelta here because zeek randomly generates uids that
		// vary in size.
		assert.InDelta(t, 1437, info.Size, 10)
		assert.Equal(t, int64(4224), info.PacketSize)
		assert.True(t, info.PacketSupport)
		assert.Equal(t, p.pcapfile, info.PacketPath)
	})
	t.Run("PacketIndexExists", func(t *testing.T) {
		require.FileExists(t, filepath.Join(p.core.Root, p.space, space.PcapIndexFile))
	})
	t.Run("TaskStartMessage", func(t *testing.T) {
		status := p.payloads[0].(*api.TaskStart)
		assert.Equal(t, status.Type, "TaskStart")
	})
	t.Run("StatusMessage", func(t *testing.T) {
		info, err := os.Stat(p.pcapfile)
		require.NoError(t, err)
		plen := len(p.payloads)
		status := p.payloads[plen-2].(*api.PacketPostStatus)
		assert.Equal(t, status.Type, "PacketPostStatus")
		assert.Equal(t, status.PacketSize, info.Size())
		assert.Equal(t, status.PacketReadSize, info.Size())
		assert.Equal(t, 1, status.SnapshotCount)
		assert.Equal(t, nano.NewSpanTs(nano.Unix(1501770877, 471635000), nano.Unix(1501770880, 988247000)), *status.Span)
	})
	t.Run("TaskEndMessage", func(t *testing.T) {
		status := p.payloads[len(p.payloads)-1].(*api.TaskEnd)
		assert.Equal(t, status.Type, "TaskEnd")
		assert.Nil(t, status.Error)
	})
}

func TestPacketPostSearch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}
	ln, err := zeek.LauncherFromPath(os.Getenv("ZEEK"))
	require.NoError(t, err)
	p := packetPost(t, "./testdata/valid.pcap", ln)
	defer p.cleanup()
	t.Run("Success", func(t *testing.T) {
		req := api.PacketSearch{
			Span:    nano.Span{Ts: 1501770877471635000, Dur: 3485852000},
			Proto:   "tcp",
			SrcHost: net.ParseIP("192.168.0.5"),
			SrcPort: 50798,
			DstHost: net.ParseIP("54.148.114.85"),
			DstPort: 80,
		}
		rc, err := p.client.PcapSearch(context.Background(), p.space, req)
		require.NoError(t, err)
		defer rc.Close()
		// just make sure it's a valid pcap
		for {
			b, _, err := rc.Read()
			require.NoError(t, err)
			if b == nil {
				return
			}
		}
	})
	t.Run("NotFound", func(t *testing.T) {
		req := api.PacketSearch{
			Span:    nano.Span{Ts: 1501770877471635000, Dur: 3485852000},
			Proto:   "tcp",
			SrcHost: net.ParseIP("192.168.0.5"),
			SrcPort: 50760,
			DstHost: net.ParseIP("54.148.114.85"),
			DstPort: 80,
		}
		_, err := p.client.PcapSearch(context.Background(), p.space, req)
		require.Equal(t, api.ErrNoPcapResultsFound, err)
	})
}

func TestPcapSearchNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}
	ln, err := zeek.LauncherFromPath(os.Getenv("ZEEK"))
	require.NoError(t, err)
	p := packetPost(t, "./testdata/valid.pcap", ln)
	defer p.cleanup()
}

func TestPacketPostInvalidPcap(t *testing.T) {
	p := packetPost(t, "./testdata/invalid.pcap", testZeekLauncher(nil, nil))
	defer p.cleanup()
	t.Run("ErrorResponse", func(t *testing.T) {
		require.Error(t, p.err)
		var reserr *api.ErrorResponse
		if !errors.As(p.err, &reserr) {
			t.Fatalf("expected error to be for type *api.ErrorResponse, got %T", p.err)
		}
		assert.Equal(t, http.StatusBadRequest, reserr.StatusCode())
		// XXX Better error message here.
		require.Regexp(t, "bad pcap file", reserr.Err.Error())
	})
	t.Run("EmptySpaceInfo", func(t *testing.T) {
		info, err := p.client.SpaceInfo(context.Background(), p.space)
		assert.NoError(t, err)
		expected := api.SpaceInfo{
			Name: p.space,
		}
		require.Equal(t, &expected, info)
	})
}

func TestPacketPostZeekFailImmediate(t *testing.T) {
	expectedErr := errors.New("zeek error: failed to start")
	startFn := func(*testZeekProcess) error { return expectedErr }
	p := packetPost(t, "./testdata/valid.pcap", testZeekLauncher(startFn, nil))
	defer p.cleanup()
	t.Run("TaskEndError", func(t *testing.T) {
		expected := &api.TaskEnd{
			Type:   "TaskEnd",
			TaskID: 1,
			Error: &api.Error{
				Type:    "Error",
				Message: expectedErr.Error(),
			},
		}
		last := p.payloads[len(p.payloads)-1]
		require.Equal(t, expected, last)
	})
}

func TestPacketPostZeekFailAfterWrite(t *testing.T) {
	expectedErr := errors.New("zeek exited after write")
	write := func(p *testZeekProcess) error {
		if err := writeLogsFn(testZeekLogs)(p); err != nil {
			return err
		}
		return expectedErr
	}
	p := packetPost(t, "./testdata/valid.pcap", testZeekLauncher(nil, write))
	defer p.cleanup()
	t.Run("TaskEndError", func(t *testing.T) {
		expected := &api.TaskEnd{
			Type:   "TaskEnd",
			TaskID: 1,
			Error: &api.Error{
				Type:    "Error",
				Message: expectedErr.Error(),
			},
		}
		last := p.payloads[len(p.payloads)-1]
		require.Equal(t, expected, last)
	})
	t.Run("EmptySpaceInfo", func(t *testing.T) {
		info, err := p.client.SpaceInfo(context.Background(), p.space)
		assert.NoError(t, err)
		expected := api.SpaceInfo{
			Name: p.space,
		}
		require.Equal(t, &expected, info)
	})
}

func packetPost(t *testing.T, pcapfile string, l zeek.Launcher) packetPostResult {
	return packetPostWithConfig(t, zqd.Config{ZeekLauncher: l}, pcapfile)
}

func packetPostWithConfig(t *testing.T, conf zqd.Config, pcapfile string) packetPostResult {
	if conf.Logger == nil {
		conf.Logger = zaptest.NewLogger(t, zaptest.Level(zapcore.WarnLevel))
	}
	c := setCoreRoot(t, conf)
	ts := httptest.NewServer(zqd.NewHandler(c, conf.Logger))
	client := api.NewConnectionTo(ts.URL)
	res := packetPostResult{
		core:     c,
		srv:      ts,
		client:   client,
		space:    "test",
		pcapfile: pcapfile,
	}
	res.postPcap(t, pcapfile)
	return res
}

func setCoreRoot(t *testing.T, c zqd.Config) *zqd.Core {
	if c.Root == "" {
		dir, err := ioutil.TempDir("", "PacketPostTest")
		require.NoError(t, err)
		c.Root = dir
	}
	if c.Logger == nil {
		c.Logger = zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	}
	return zqd.NewCore(c)
}

type packetPostResult struct {
	core     *zqd.Core
	client   *api.Connection
	srv      *httptest.Server
	space    string
	pcapfile string
	body     []byte
	err      error
	payloads []interface{}
}

func (r *packetPostResult) postPcap(t *testing.T, file string) {
	_, err := r.client.SpacePost(context.Background(), api.SpacePostRequest{Name: r.space})
	require.NoError(t, err)
	var stream *api.Stream
	stream, r.err = r.client.PacketPost(context.Background(), r.space, api.PacketPostRequest{r.pcapfile})
	if r.err == nil {
		r.readPayloads(t, stream)
	}
}

func (r *packetPostResult) readPayloads(t *testing.T, stream *api.Stream) {
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
	r.srv.Close()
}
