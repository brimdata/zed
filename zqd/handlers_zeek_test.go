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
	"github.com/brimsec/zq/zqd/storage"
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

func TestPcapPostSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}
	ln, err := zeek.LauncherFromPath(os.Getenv("ZEEK"))
	require.NoError(t, err)
	p := pcapPost(t, "./testdata/valid.pcap", ln)
	defer p.cleanup()
	t.Run("DataReverseSorted", func(t *testing.T) {
		expected := `
#0:record[ts:time]
0:[1501770880.988247;]
0:[1501770877.501001;]
0:[1501770877.471635;]
0:[1501770877.471635;]`
		res := searchTzng(t, p.client, p.space.ID, "cut ts")
		assert.Equal(t, test.Trim(expected), res)
	})
	t.Run("SpaceInfo", func(t *testing.T) {
		info, err := p.client.SpaceInfo(context.Background(), p.space.ID)
		assert.NoError(t, err)
		assert.Equal(t, p.space.ID, info.ID)
		assert.Equal(t, nano.NewSpanTs(nano.Unix(1501770877, 471635000), nano.Unix(1501770880, 988247001)), *info.Span)
		// Must use InDelta here because zeek randomly generates uids that
		// vary in size.
		assert.InDelta(t, 1437, info.Size, 10)
		assert.Equal(t, int64(4224), info.PcapSize)
		assert.True(t, info.PcapSupport)
		assert.Equal(t, p.pcapfile, info.PcapPath)
	})
	t.Run("PcapIndexExists", func(t *testing.T) {
		require.FileExists(t, filepath.Join(p.core.Root, string(p.space.ID), space.PcapIndexFile))
	})
	t.Run("TaskStartMessage", func(t *testing.T) {
		status := p.payloads[0].(*api.TaskStart)
		assert.Equal(t, status.Type, "TaskStart")
	})
	t.Run("StatusMessage", func(t *testing.T) {
		info, err := os.Stat(p.pcapfile)
		require.NoError(t, err)
		plen := len(p.payloads)
		status := p.payloads[plen-2].(*api.PcapPostStatus)
		assert.Equal(t, status.Type, "PcapPostStatus")
		assert.Equal(t, status.PcapSize, info.Size())
		assert.Equal(t, status.PcapReadSize, info.Size())
		assert.Equal(t, 1, status.SnapshotCount)
		assert.Equal(t, nano.NewSpanTs(nano.Unix(1501770877, 471635000), nano.Unix(1501770880, 988247001)), *status.Span)
	})
	t.Run("TaskEndMessage", func(t *testing.T) {
		status := p.payloads[len(p.payloads)-1].(*api.TaskEnd)
		assert.Equal(t, status.Type, "TaskEnd")
		assert.Nil(t, status.Error)
	})
}

func TestPcapPostSearch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}
	ln, err := zeek.LauncherFromPath(os.Getenv("ZEEK"))
	require.NoError(t, err)
	p := pcapPost(t, "./testdata/valid.pcap", ln)
	defer p.cleanup()
	t.Run("Success", func(t *testing.T) {
		req := api.PcapSearch{
			Span:    nano.Span{Ts: 1501770877471635000, Dur: 3485852000},
			Proto:   "tcp",
			SrcHost: net.ParseIP("192.168.0.5"),
			SrcPort: 50798,
			DstHost: net.ParseIP("54.148.114.85"),
			DstPort: 80,
		}
		rc, err := p.client.PcapSearch(context.Background(), p.space.ID, req)
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
		req := api.PcapSearch{
			Span:    nano.Span{Ts: 1501770877471635000, Dur: 3485852000},
			Proto:   "tcp",
			SrcHost: net.ParseIP("192.168.0.5"),
			SrcPort: 50760,
			DstHost: net.ParseIP("54.148.114.85"),
			DstPort: 80,
		}
		_, err := p.client.PcapSearch(context.Background(), p.space.ID, req)
		require.Equal(t, api.ErrNoPcapResultsFound, err)
	})
}

func TestPcapSearchNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test for windows")
	}
	ln, err := zeek.LauncherFromPath(os.Getenv("ZEEK"))
	require.NoError(t, err)
	p := pcapPost(t, "./testdata/valid.pcap", ln)
	defer p.cleanup()
}

func TestPcapPostInvalidPcap(t *testing.T) {
	p := pcapPost(t, "./testdata/invalid.pcap", testZeekLauncher(nil, nil))
	defer p.cleanup()
	t.Run("ErrorResponse", func(t *testing.T) {
		require.Error(t, p.err)
		var reserr *api.ErrorResponse
		if !errors.As(p.err, &reserr) {
			t.Fatalf("expected error to be for type *api.ErrorResponse, got %T", p.err)
		}
		assert.Equal(t, http.StatusBadRequest, reserr.StatusCode())
		require.Regexp(t, "invalid pcap: pcap: unknown magic 73696874; pcapng: first block type not a section header: 1936287860", reserr.Err.Error())
	})
	t.Run("EmptySpaceInfo", func(t *testing.T) {
		info, err := p.client.SpaceInfo(context.Background(), p.space.ID)
		assert.NoError(t, err)
		expected := api.SpaceInfo{
			ID:          p.space.ID,
			Name:        p.space.Name,
			DataPath:    p.space.DataPath,
			StorageKind: storage.FileStore.String(),
		}
		require.Equal(t, &expected, info)
	})
}

func TestPcapPostZeekFailImmediate(t *testing.T) {
	expectedErr := errors.New("zeek error: failed to start")
	startFn := func(*testZeekProcess) error { return expectedErr }
	p := pcapPost(t, "./testdata/valid.pcap", testZeekLauncher(startFn, nil))
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

func TestPcapPostZeekFailAfterWrite(t *testing.T) {
	expectedErr := errors.New("zeek exited after write")
	write := func(p *testZeekProcess) error {
		if err := writeLogsFn(testZeekLogs)(p); err != nil {
			return err
		}
		return expectedErr
	}
	p := pcapPost(t, "./testdata/valid.pcap", testZeekLauncher(nil, write))
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
		info, err := p.client.SpaceInfo(context.Background(), p.space.ID)
		assert.NoError(t, err)
		expected := api.SpaceInfo{
			ID:          p.space.ID,
			Name:        p.space.Name,
			DataPath:    p.space.DataPath,
			StorageKind: storage.FileStore.String(),
		}
		require.Equal(t, &expected, info)
	})
}

func pcapPost(t *testing.T, pcapfile string, l zeek.Launcher) pcapPostResult {
	return pcapPostWithConfig(t, zqd.Config{ZeekLauncher: l}, pcapfile)
}

func pcapPostWithConfig(t *testing.T, conf zqd.Config, pcapfile string) pcapPostResult {
	if conf.Logger == nil {
		conf.Logger = zaptest.NewLogger(t, zaptest.Level(zapcore.WarnLevel))
	}
	c := setCoreRoot(t, conf)
	ts := httptest.NewServer(zqd.NewHandler(c, conf.Logger))
	client := api.NewConnectionTo(ts.URL)
	sp, err := client.SpacePost(context.Background(), api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	res := pcapPostResult{
		core:     c,
		srv:      ts,
		client:   client,
		space:    *sp,
		pcapfile: pcapfile,
	}
	res.postPcap(t, pcapfile)
	return res
}

func setCoreRoot(t *testing.T, c zqd.Config) *zqd.Core {
	if c.Root == "" {
		dir, err := ioutil.TempDir("", "PcapPostTest")
		require.NoError(t, err)
		c.Root = dir
	}
	if c.Logger == nil {
		c.Logger = zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	}
	core, err := zqd.NewCore(c)
	require.NoError(t, err)
	return core
}

type pcapPostResult struct {
	core     *zqd.Core
	client   *api.Connection
	srv      *httptest.Server
	space    api.SpaceInfo
	pcapfile string
	body     []byte
	err      error
	payloads []interface{}
}

func (r *pcapPostResult) postPcap(t *testing.T, file string) {
	var stream *api.Stream
	stream, r.err = r.client.PcapPost(context.Background(), r.space.ID, api.PcapPostRequest{r.pcapfile})
	if r.err == nil {
		r.readPayloads(t, stream)
	}
}

func (r *pcapPostResult) readPayloads(t *testing.T, stream *api.Stream) {
	for {
		i, err := stream.Next()
		require.NoError(t, err)
		if i == nil {
			break
		}
		r.payloads = append(r.payloads, i)
	}
}

func (r *pcapPostResult) cleanup() {
	os.RemoveAll(r.core.Root)
	r.srv.Close()
}
