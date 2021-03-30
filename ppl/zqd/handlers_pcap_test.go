// +build pcapingest

package zqd_test

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/pkg/fs"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/test"
	"github.com/brimdata/zed/ppl/zqd"
	"github.com/brimdata/zed/ppl/zqd/pcapanalyzer"
	"github.com/brimdata/zed/ppl/zqd/pcapstorage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	const pcapfile = "testdata/valid.pcap"
	p := pcapPostTest(t, pcapfile, launcherFromEnv(t, "ZEEK"))
	t.Run("DataReverseSorted", func(t *testing.T) {
		expected := `
{ts:2017-08-03T14:34:40.988247Z}
{ts:2017-08-03T14:34:40.988247Z}
{ts:2017-08-03T14:34:37.501001Z}
{ts:2017-08-03T14:34:37.471635Z}
{ts:2017-08-03T14:34:37.471635Z}
`
		res := searchZson(t, p.client, p.space.ID, "pick ts")
		assert.Equal(t, test.Trim(expected), res)
	})
	t.Run("SpaceInfo", func(t *testing.T) {
		info, err := p.client.SpaceInfo(context.Background(), p.space.ID)
		assert.NoError(t, err)
		assert.Equal(t, info.ID, p.space.ID)
		assert.Equal(t, nano.NewSpanTs(nano.Unix(1501770877, 471635000), nano.Unix(1501770880, 988247001)), *info.Span)
		// Must use InDelta here because zeek randomly generates uids that
		// vary in size.
		assert.InDelta(t, 1561, info.Size, 10)
		assert.Equal(t, int64(4224), info.PcapSize)
		assert.True(t, info.PcapSupport)
		assert.Equal(t, iosrc.MustParseURI(pcapfile), info.PcapPath)
	})
	t.Run("PcapIndexExists", func(t *testing.T) {
		require.FileExists(t, p.core.Root().AppendPath(string(p.space.ID), pcapstorage.MetaFile).Filepath())
	})
	t.Run("TaskStartMessage", func(t *testing.T) {
		status := p.payloads[0].(*api.TaskStart)
		assert.Equal(t, status.Type, "TaskStart")
	})
	t.Run("StatusMessage", func(t *testing.T) {
		info, err := os.Stat(pcapfile)
		require.NoError(t, err)
		plen := len(p.payloads)
		status := p.payloads[plen-2].(*api.PcapPostStatus)
		assert.Equal(t, "PcapPostStatus", status.Type)
		assert.Equal(t, info.Size(), status.PcapSize)
		assert.Equal(t, info.Size(), status.PcapReadSize)
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
	const pcapfile = "testdata/valid.pcap"
	p := pcapPostTest(t, pcapfile, launcherFromEnv(t, "ZEEK"))
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
		require.Equal(t, client.ErrNoPcapResultsFound, err)
	})
}

func TestPcapPostPcapNgWithExtraBytes(t *testing.T) {
	p := pcapPostTest(t, "testdata/extra.pcapng", testLauncher(nil, nil))
	t.Run("PcapNgExtra", func(t *testing.T) {
		require.NoError(t, p.err)
		warning := p.payloads[1].(*api.PcapPostWarning)
		assert.Equal(t, "pcap-ng has extra bytes at eof: 20", warning.Warning)
	})
}

func TestPcapPostInvalidPcap(t *testing.T) {
	p := pcapPostTest(t, "testdata/invalid.pcap", testLauncher(nil, nil))
	t.Run("ErrorResponse", func(t *testing.T) {
		require.Error(t, p.err)
		var reserr *client.ErrorResponse
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
			Space: api.Space{
				ID:          p.space.ID,
				Name:        p.space.Name,
				DataPath:    p.space.DataPath,
				StorageKind: api.DefaultStorageKind(),
			},
		}
		require.Equal(t, &expected, info)
	})
}

func TestPcapPostZeekFailImmediate(t *testing.T) {
	expectedErr := errors.New("zeek error: failed to start")
	startFn := func(*testPcapProcess) error { return expectedErr }
	p := pcapPostTest(t, "testdata/valid.pcap", testLauncher(startFn, nil))
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
	write := func(p *testPcapProcess) error {
		if err := writeLogsFn(testZeekLogs)(p); err != nil {
			return err
		}
		return expectedErr
	}
	p := pcapPostTest(t, "testdata/valid.pcap", testLauncher(nil, write))
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

// TestPcapTimeRange verifies that the time range for a space with an imported
// pcap includes the pcap's time range, regardless of whether any records are
// present or not. See issue 1797.
func TestPcapTimeRange(t *testing.T) {
	noopWrite := func(p *testPcapProcess) error {
		_, err := ioutil.ReadAll(p.reader)
		return err
	}
	p := pcapPostTest(t, "testdata/valid.pcap", testLauncher(nil, noopWrite))
	info, err := p.client.SpaceInfo(context.Background(), p.space.ID)
	require.NoError(t, err)
	require.NotNil(t, info.Span)
	exp := nano.NewSpanTs(nano.Unix(1501770877, 471635000), nano.Unix(1501770880, 988247000))
	require.Equal(t, exp, *info.Span)
}

func launcherFromEnv(t *testing.T, key string) pcapanalyzer.Launcher {
	ln, err := pcapanalyzer.LauncherFromPath(os.Getenv(key), false)
	require.NoError(t, err)
	return ln
}

func testLauncher(start, wait procFn) pcapanalyzer.Launcher {
	return func(ctx context.Context, r io.Reader, dir string) (pcapanalyzer.ProcessWaiter, error) {
		p := &testPcapProcess{
			ctx:    ctx,
			reader: r,
			wd:     dir,
			wait:   wait,
			start:  start,
		}
		return p, p.Start()
	}
}

type testPcapProcess struct {
	ctx    context.Context
	reader io.Reader
	wd     string
	start  procFn
	wait   procFn
}

func (p *testPcapProcess) Start() error {
	if p.start != nil {
		return p.start(p)
	}
	return nil
}

func (p *testPcapProcess) Wait() error {
	if p.wait != nil {
		return p.wait(p)
	}
	_, err := ioutil.ReadAll(p.reader)
	return err
}

func (p *testPcapProcess) Stdout() string { return "" }

type procFn func(t *testPcapProcess) error

func writeLogsFn(logs []string) procFn {
	return func(p *testPcapProcess) error {
		for _, log := range logs {
			r, err := fs.Open(log)
			if err != nil {
				return err
			}
			defer r.Close()
			base := filepath.Base(r.Name())
			w, err := os.Create(filepath.Join(p.wd, base))
			if err != nil {
				return err
			}
			defer w.Close()
			if _, err = io.Copy(w, r); err != nil {
				return err
			}
		}
		// drain the reader
		_, err := io.Copy(ioutil.Discard, p.reader)
		return err
	}
}

type pcapPostTestResult struct {
	client   *client.Connection
	core     *zqd.Core
	space    api.Space
	err      error
	payloads client.Payloads
}

func pcapPostTest(t *testing.T, pcapfile string, zeek pcapanalyzer.Launcher) pcapPostTestResult {
	return testPcapPostWithConfig(t, zqd.Config{Zeek: zeek}, pcapfile)
}

func testPcapPostWithConfig(t *testing.T, conf zqd.Config, pcapfile string) pcapPostTestResult {
	ctx := context.Background()
	c, client := newCoreWithConfig(t, conf)
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	payloads, perr := client.PcapPost(ctx, sp.ID, api.PcapPostRequest{pcapfile})
	return pcapPostTestResult{
		err:      perr,
		payloads: payloads,
		space:    *sp,
		core:     c,
		client:   client,
	}
}
