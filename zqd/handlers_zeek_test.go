// +build zeek

package zqd_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	"github.com/brimsec/zq/zqd/ingest"
	"github.com/brimsec/zq/zqd/zeek"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
		assert.Equal(t, int64(4224), info.PacketSize)
		assert.True(t, info.PacketSupport)
		assert.Equal(t, p.pcapfile, info.PacketPath)
	})
	t.Run("PacketIndexExists", func(t *testing.T) {
		require.FileExists(t, filepath.Join(p.core.Root, p.space, ingest.PcapIndexFile))
	})
	t.Run("TaskStartMessage", func(t *testing.T) {
		status := p.payloads[0].(*api.TaskStart)
		assert.Equal(t, status.Type, "TaskStart")
	})
	t.Run("StatusMessage", func(t *testing.T) {
		info, err := os.Stat(p.pcapfile)
		require.NoError(t, err)
		status := p.payloads[1].(*api.PacketPostStatus)
		assert.Equal(t, status.Type, "PacketPostStatus")
		assert.Equal(t, status.PacketSize, info.Size())
		assert.Equal(t, status.PacketReadSize, info.Size())
		assert.Equal(t, 1, status.SnapshotCount)
		assert.Equal(t, nano.Unix(1501770877, 471635000), *status.MinTime)
		assert.Equal(t, nano.Unix(1501770880, 988247000), *status.MaxTime)
	})
	t.Run("TaskEndMessage", func(t *testing.T) {
		status := p.payloads[len(p.payloads)-1].(*api.TaskEnd)
		assert.Equal(t, status.Type, "TaskEnd")
		assert.Nil(t, status.Error)
	})
}

func TestPacketPostSortLimit(t *testing.T) {
	fn := writeLogsFn(testZeekLogs)
	ln := testZeekLauncher(nil, fn)
	p := packetPostWithConfig(t, zqd.Config{SortLimit: 1, ZeekLauncher: ln}, "./testdata/valid.pcap")
	defer p.cleanup()
	t.Run("TaskEndError", func(t *testing.T) {
		taskEnd := p.payloads[len(p.payloads)-1].(*api.TaskEnd)
		assert.Equal(t, "TaskEnd", taskEnd.Type)
		assert.NotNil(t, taskEnd.Error)
		assert.Regexp(t, "sort limit", taskEnd.Error.Message)
	})
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
		require.Regexp(t, "^bad pcap file*", reserr.Err.Error())
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
		u := fmt.Sprintf("http://localhost:9867/space/%s", p.space)
		var info api.SpaceInfo
		httpJSONSuccess(t, zqd.NewHandler(p.core), "GET", u, nil, &info)
		expected := api.SpaceInfo{
			Name: p.space,
		}
		require.Equal(t, expected, info)
	})
}

func packetPost(t *testing.T, pcapfile string, l zeek.Launcher) packetPostResult {
	return packetPostWithConfig(t, zqd.Config{ZeekLauncher: l}, pcapfile)
}

func packetPostWithConfig(t *testing.T, conf zqd.Config, pcapfile string) packetPostResult {
	c := setCoreRoot(t, conf)
	ts := httptest.NewServer(zqd.NewHandler(c))
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
	stream, r.err = r.client.PostPacket(context.Background(), r.space, api.PacketPostRequest{r.pcapfile})
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

func testZeekLauncher(start, wait procFn) zeek.Launcher {
	return func(ctx context.Context, r io.Reader, dir string) (zeek.Process, error) {
		p := &testZeekProcess{
			ctx:    ctx,
			reader: r,
			wd:     dir,
			wait:   wait,
			start:  start,
		}
		return p, p.Start()
	}
}

type procFn func(t *testZeekProcess) error

type testZeekProcess struct {
	ctx    context.Context
	reader io.Reader
	wd     string
	start  procFn
	wait   procFn
}

func (p *testZeekProcess) Start() error {
	if p.start != nil {
		return p.start(p)
	}
	return nil
}

func (p *testZeekProcess) Wait() error {
	if p.wait != nil {
		return p.wait(p)
	}
	return nil
}

func writeLogsFn(logs []string) procFn {
	return func(t *testZeekProcess) error {
		for _, log := range logs {
			r, err := os.Open(log)
			if err != nil {
				return err
			}
			defer r.Close()
			base := filepath.Base(r.Name())
			w, err := os.Create(filepath.Join(t.wd, base))
			if err != nil {
				return err
			}
			defer w.Close()
			if _, err = io.Copy(w, r); err != nil {
				return err
			}
		}
		// drain the reader
		_, err := io.Copy(ioutil.Discard, t.reader)
		return err
	}
}
