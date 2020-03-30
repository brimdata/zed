package zqd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestSearch(t *testing.T) {
	src := `
#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]
`
	core, client, done := newCore(t)
	defer done()
	sp, err := client.SpacePost(context.Background(), api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	putSpaceData(t, core, sp.Name, src)
	res := zngSearch(t, client, sp.Name, "*")
	require.Equal(t, test.Trim(src), res)
}

func TestSearchEmptySpace(t *testing.T) {
	space := "test"
	ctx := context.Background()
	_, client, done := newCore(t)
	defer done()
	_, err := client.SpacePost(ctx, api.SpacePostRequest{Name: space})
	require.NoError(t, err)
	res := zngSearch(t, client, space, "*")
	require.Equal(t, "", res)
}

func TestSpaceList(t *testing.T) {
	ctx := context.Background()
	c, client, done := newCore(t)
	defer done()
	sp1, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "sp1"})
	require.NoError(t, err)
	sp2, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "sp2"})
	require.NoError(t, err)
	sp3, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "sp3"})
	require.NoError(t, err)
	sp4, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "sp4"})
	require.NoError(t, err)
	// delete config.json from sp3
	require.NoError(t, os.Remove(filepath.Join(c.Root, sp3.Name, "config.json")))
	expected := []string{
		sp1.Name,
		sp2.Name,
		sp4.Name,
	}
	list, err := client.SpaceList(ctx)
	require.NoError(t, err)
	require.Equal(t, expected, list)
}

func TestSpaceInfo(t *testing.T) {
	src := `
#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]`
	ctx := context.Background()
	c, client, done := newCore(t)
	defer done()
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	putSpaceData(t, c, sp.Name, src)
	expected := &api.SpaceInfo{
		// MinTime and MaxTime are not present because the
		// space is not populated via the regular pcap ingest
		// process.
		Name:          sp.Name,
		Size:          88,
		PacketSupport: false,
	}
	info, err := client.SpaceInfo(ctx, sp.Name)
	require.NoError(t, err)
	require.Equal(t, expected, info)
}

func TestSpaceInfoNoData(t *testing.T) {
	ctx := context.Background()
	_, client, done := newCore(t)
	defer done()
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	info, err := client.SpaceInfo(ctx, sp.Name)
	require.NoError(t, err)
	expected := &api.SpaceInfo{
		Name:          sp.Name,
		Size:          0,
		PacketSupport: false,
	}
	require.Equal(t, expected, info)
}

func TestSpacePostNameOnly(t *testing.T) {
	ctx := context.Background()
	c, client, done := newCore(t)
	defer done()
	expected := &api.SpacePostResponse{
		Name:    "test",
		DataDir: filepath.Join(c.Root, "test"),
	}
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	require.Equal(t, expected, sp)
}

func TestSpacePostDataDir(t *testing.T) {
	ctx := context.Background()
	tmp := createTempDir(t)
	defer os.RemoveAll(tmp)
	datadir := filepath.Join(tmp, "mypcap.brim")
	expected := &api.SpacePostResponse{
		Name:    "mypcap.brim",
		DataDir: datadir,
	}
	_, client, done := newCoreAtDir(t, filepath.Join(tmp, "spaces"))
	defer done()
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{DataDir: datadir})
	require.NoError(t, err)
	require.Equal(t, expected, sp)
}

func TestSpacePostDataDirDuplicate(t *testing.T) {
	ctx := context.Background()
	tmp1 := createTempDir(t)
	defer os.RemoveAll(tmp1)
	tmp2 := createTempDir(t)
	defer os.RemoveAll(tmp2)
	datadir1 := filepath.Join(tmp1, "mypcap.brim")
	datadir2 := filepath.Join(tmp2, "mypcap.brim")
	expected := &api.SpacePostResponse{
		Name:    "mypcap_01.brim",
		DataDir: datadir2,
	}
	_, client, done := newCoreAtDir(t, filepath.Join(tmp1, "spaces"))
	defer done()
	_, err := client.SpacePost(ctx, api.SpacePostRequest{DataDir: datadir1})
	require.NoError(t, err)
	sp2, err := client.SpacePost(ctx, api.SpacePostRequest{DataDir: datadir2})
	require.NoError(t, err)
	require.Equal(t, expected, sp2)
}

func TestSpaceDelete(t *testing.T) {
	ctx := context.Background()
	_, client, done := newCore(t)
	defer done()
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	err = client.SpaceDelete(ctx, sp.Name)
	require.NoError(t, err)
	list, err := client.SpaceList(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{}, list)
}

func TestSpaceDeleteDataDir(t *testing.T) {
	ctx := context.Background()
	tmp := createTempDir(t)
	defer os.RemoveAll(tmp)
	_, client, done := newCoreAtDir(t, filepath.Join(tmp, "spaces"))
	defer done()
	datadir := filepath.Join(tmp, "mypcap.brim")
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{Name: "test"})
	require.NoError(t, err)
	err = client.SpaceDelete(ctx, sp.Name)
	require.NoError(t, err)
	list, err := client.SpaceList(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{}, list)
	// ensure data dir has also been deleted
	_, err = os.Stat(datadir)
	require.Error(t, err)
	require.Truef(t, os.IsNotExist(err), "expected error to be os.IsNotExist, got %v", err)
}

func TestURLEncodingSupport(t *testing.T) {
	ctx := context.Background()
	_, client, done := newCore(t)
	defer done()
	rawSpace := "raw %space.brim"
	sp, err := client.SpacePost(ctx, api.SpacePostRequest{Name: rawSpace})
	require.NoError(t, err)
	require.Equal(t, rawSpace, sp.Name)
	err = client.SpaceDelete(ctx, rawSpace)
	require.NoError(t, err)
}

func TestNoEndSlashSupport(t *testing.T) {
	_, client, done := newCore(t)
	defer done()
	_, err := client.Do(context.Background(), "GET", "/space/", nil)
	require.Error(t, err)
	require.Equal(t, 404, err.(*api.ErrorResponse).StatusCode())
}

func TestRequestID(t *testing.T) {
	ctx := context.Background()
	t.Run("GeneratesUniqueID", func(t *testing.T) {
		_, client, done := newCore(t)
		defer done()
		res1, err := client.Do(ctx, "GET", "/space", nil)
		require.NoError(t, err)
		res2, err := client.Do(ctx, "GET", "/space", nil)
		require.NoError(t, err)
		assert.Equal(t, "1", res1.Header().Get("X-Request-ID"))
		assert.Equal(t, "2", res2.Header().Get("X-Request-ID"))
	})
	t.Run("PropagatesID", func(t *testing.T) {
		_, client, done := newCore(t)
		defer done()
		requestID := "random-request-ID"
		req := client.Request(context.Background())
		req.SetHeader("X-Request-ID", requestID)
		res, err := req.Execute("GET", "/space")
		require.NoError(t, err)
		require.Equal(t, requestID, res.Header().Get("X-Request-ID"))
	})
}

func zngSearch(t *testing.T, client *api.Connection, space, prog string) string {
	parsed, err := zql.ParseProc(prog)
	require.NoError(t, err)
	proc, err := json.Marshal(parsed)
	require.NoError(t, err)
	req := api.SearchRequest{
		Space: space,
		Proc:  proc,
		Span:  nano.MaxSpan,
		Dir:   1,
	}
	r, err := client.Search(context.Background(), req)
	require.NoError(t, err)
	buf := bytes.NewBuffer(nil)
	w := zbuf.NopFlusher(zngio.NewWriter(buf))
	require.NoError(t, zbuf.Copy(w, r))
	return buf.String()
}

func createTempDir(t *testing.T) string {
	// Replace '/' with '-' so it doesn't try to access dirs that don't exist.
	// Apparently "/" in a filepath for windows still tries to create a
	// directory; this solution works for windows as well.
	name := strings.ReplaceAll(t.Name(), "/", "-")
	dir, err := ioutil.TempDir("", name)
	require.NoError(t, err)
	return dir
}

func newCore(t *testing.T) (*zqd.Core, *api.Connection, func()) {
	root := createTempDir(t)
	return newCoreAtDir(t, root)
}

func newCoreAtDir(t *testing.T, dir string) (*zqd.Core, *api.Connection, func()) {
	conf := zqd.Config{
		Root:   dir,
		Logger: zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)),
	}
	require.NoError(t, os.MkdirAll(dir, 0755))
	c := zqd.NewCore(conf)
	h := zqd.NewHandler(c)
	ts := httptest.NewServer(h)
	return c, api.NewConnectionTo(ts.URL), func() {
		os.RemoveAll(dir)
		ts.Close()
	}
}

func createSpace(t *testing.T, c *zqd.Core, spaceName, datadir string) api.SpacePostResponse {
	req := api.SpacePostRequest{
		Name:    spaceName,
		DataDir: datadir,
	}
	var res api.SpacePostResponse
	httpJSONSuccess(t, zqd.NewHandler(c), "POST", "http://localhost:9867/space", req, &res)
	return res
}

// putSpaceData writes the provided zng source in to the provided space
// directory.
func putSpaceData(t *testing.T, c *zqd.Core, spaceName, src string) {
	s, err := space.Open(c.Root, spaceName)
	require.NoError(t, err)
	f, err := s.CreateFile("all.bzng")
	require.NoError(t, err)
	defer f.Close()
	// write zng
	w := bzngio.NewWriter(f)
	r := zngio.NewReader(strings.NewReader(src), resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
}

func httpRequest(t *testing.T, h http.Handler, method, url string, body interface{}) *http.Response {
	var rw io.ReadWriter
	if body != nil {
		rw = bytes.NewBuffer(nil)
		if err := json.NewEncoder(rw).Encode(body); err != nil {
			panic(err)
		}
	}
	r := httptest.NewRequest(method, url, rw)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Result()
}

func httpJSONSuccess(t *testing.T, h http.Handler, method, url string, body interface{}, res interface{}) {
	r := httpRequest(t, h, method, url, body)
	if r.StatusCode < 200 || r.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(r.Body)
		require.Equal(t, http.StatusOK, r.StatusCode, string(body))
	}
	require.Equal(t, "application/json", r.Header.Get("Content-Type"))
	require.NoError(t, json.NewDecoder(r.Body).Decode(res))
}
