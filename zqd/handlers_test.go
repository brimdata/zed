package zqd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestSimpleSearch(t *testing.T) {
	src := `
#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]
`
	space := "test"
	c := newCore(t)
	defer os.RemoveAll(c.Root)
	createSpaceWithData(t, c, space, src)
	require.Equal(t, test.Trim(src), execSearch(t, c, space, "*"))
}

func TestSearchEmptySpace(t *testing.T) {
	space := "test"
	c := newCore(t)
	defer os.RemoveAll(c.Root)
	createSpace(t, c, space, "")
	require.Equal(t, "", execSearch(t, c, space, "*"))
}

func TestSpaceList(t *testing.T) {
	c := newCore(t)
	defer os.RemoveAll(c.Root)
	sp1 := createSpace(t, c, "sp1", "")
	sp2 := createSpace(t, c, "sp2", "")
	sp3 := createSpace(t, c, "sp3", "")
	sp4 := createSpace(t, c, "sp4", "")
	// delete config.json from sp3
	require.NoError(t, os.Remove(filepath.Join(c.Root, sp3.Name, "config.json")))
	expected := []string{
		sp1.Name,
		sp2.Name,
		sp4.Name,
	}
	var list []string
	httpJSONSuccess(t, zqd.NewHandler(c), "GET", "http://localhost:9867/space", nil, &list)
	require.Equal(t, expected, list)
}

func TestSpaceInfo(t *testing.T) {
	space := "test"
	src := `
#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]`
	c := newCore(t)
	defer os.RemoveAll(c.Root)
	createSpaceWithData(t, c, space, src)
	expected := api.SpaceInfo{
		// MinTime and MaxTime are not present because the
		// space is not populated via the regular pcap ingest
		// process.
		Name:          space,
		Size:          88,
		PacketSupport: false,
	}
	u := fmt.Sprintf("http://localhost:9867/space/%s", space)
	var info api.SpaceInfo
	httpJSONSuccess(t, zqd.NewHandler(c), "GET", u, nil, &info)
	require.Equal(t, expected, info)
}

func TestSpaceInfoNoData(t *testing.T) {
	const space = "test"
	c := newCore(t)
	createSpace(t, c, space, "")
	u := fmt.Sprintf("http://localhost:9867/space/%s", space)
	var info api.SpaceInfo
	httpJSONSuccess(t, zqd.NewHandler(c), "GET", u, nil, &info)
	expected := api.SpaceInfo{
		Name:          space,
		Size:          0,
		PacketSupport: false,
	}
	require.Equal(t, expected, info)
}

func TestSpacePostNameOnly(t *testing.T) {
	c := newCore(t)
	defer os.RemoveAll(c.Root)
	expected := api.SpacePostResponse{
		Name:    "test",
		DataDir: filepath.Join(c.Root, "test"),
	}
	res := createSpace(t, c, "test", "")
	require.Equal(t, expected, res)
}

func TestSpacePostDataDirOnly(t *testing.T) {
	run := func(name string, cb func(*testing.T, string, *zqd.Core) (string, api.SpacePostResponse)) {
		tmp := createTempDir(t)
		defer os.RemoveAll(tmp)
		c := newCoreAtDir(t, filepath.Join(tmp, "spaces"))
		require.NoError(t, os.Mkdir(c.Root, 0755))
		t.Run(name, func(t *testing.T) {
			datadir, expected := cb(t, tmp, c)
			res := createSpace(t, c, "", datadir)
			require.Equal(t, expected, res)
		})
	}
	run("Simple", func(t *testing.T, tmp string, c *zqd.Core) (string, api.SpacePostResponse) {
		datadir := filepath.Join(tmp, "mypcap.brim")
		require.NoError(t, os.Mkdir(datadir, 0755))
		return datadir, api.SpacePostResponse{
			Name:    "mypcap.brim",
			DataDir: datadir,
		}
	})
	run("DuplicateName", func(t *testing.T, tmp string, c *zqd.Core) (string, api.SpacePostResponse) {
		createSpace(t, c, "mypcap.brim", "")
		datadir := filepath.Join(tmp, "mypcap.brim")
		require.NoError(t, os.Mkdir(datadir, 0755))
		return datadir, api.SpacePostResponse{
			Name:    "mypcap_01.brim",
			DataDir: datadir,
		}
	})
}

func TestSpaceDelete(t *testing.T) {
	space := "myspace"
	spaceUrl := fmt.Sprintf("http://localhost:9867/space/%s", space)
	run := func(name string, cb func(*testing.T, string, *zqd.Core)) {
		tmp := createTempDir(t)
		defer os.RemoveAll(tmp)
		c := newCoreAtDir(t, filepath.Join(tmp, "spaces"))
		require.NoError(t, os.Mkdir(c.Root, 0755))
		t.Run(name, func(t *testing.T) {
			cb(t, tmp, c)
			// make sure no spaces exist
			var list []string
			httpJSONSuccess(t, zqd.NewHandler(c), "GET", "http://localhost:9867/space", nil, &list)
			require.Equal(t, []string{}, list)
		})
	}
	run("Simple", func(t *testing.T, tmp string, c *zqd.Core) {
		createSpace(t, c, space, "")
		r := httpRequest(t, zqd.NewHandler(c), "DELETE", spaceUrl, nil)
		require.Equal(t, http.StatusNoContent, r.StatusCode)
	})
	run("DeletesOutsideDataDir", func(t *testing.T, tmp string, c *zqd.Core) {
		datadir := filepath.Join(tmp, "datadir")
		createSpace(t, c, space, datadir)
		r := httpRequest(t, zqd.NewHandler(c), "DELETE", spaceUrl, nil)
		require.Equal(t, http.StatusNoContent, r.StatusCode)
		_, err := os.Stat(datadir)
		require.Error(t, err)
		require.Truef(t, os.IsNotExist(err), "expected error to be os.IsNotExist, got %v", err)
	})
}

func TestURLEncodingSupport(t *testing.T) {
	c := newCore(t)
	defer os.RemoveAll(c.Root)

	rawSpace := "raw %space.brim"
	encodedSpaceURL := fmt.Sprintf("http://localhost:9867/space/%s", url.PathEscape(rawSpace))

	createSpace(t, c, rawSpace, "")

	res := httpRequest(t, zqd.NewHandler(c), "GET", encodedSpaceURL, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)

	res = httpRequest(t, zqd.NewHandler(c), "DELETE", encodedSpaceURL, nil)
	require.Equal(t, http.StatusNoContent, res.StatusCode)
}

func TestNoEndSlashSupport(t *testing.T) {
	c := newCore(t)
	defer os.RemoveAll(c.Root)

	h := zqd.NewHandler(c)
	r := httptest.NewRequest("GET", "http://localhost:9867/space/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	res := w.Result()
	require.Equal(t, 404, res.StatusCode)
}

func TestRequestID(t *testing.T) {
	t.Run("GeneratesUniqueID", func(t *testing.T) {
		c := newCore(t)
		defer os.RemoveAll(c.Root)
		h := zqd.NewHandler(c)
		res1 := httpRequest(t, h, "GET", "http://localhost:9867/space", nil)
		res2 := httpRequest(t, h, "GET", "http://localhost:9867/space", nil)
		assert.Equal(t, "1", res1.Header.Get("X-Request-ID"))
		assert.Equal(t, "2", res2.Header.Get("X-Request-ID"))
	})
	t.Run("PropagatesID", func(t *testing.T) {
		c := newCore(t)
		defer os.RemoveAll(c.Root)
		requestID := "random-request-ID"
		r := httptest.NewRequest("GET", "http://localhost:9867/space", nil)
		r.Header.Add("X-Request-ID", requestID)
		w := httptest.NewRecorder()
		zqd.NewHandler(c).ServeHTTP(w, r)
		require.Equal(t, requestID, w.Result().Header.Get("X-Request-ID"))
	})
}

func execSearch(t *testing.T, c *zqd.Core, space, prog string) string {
	parsed, err := zql.ParseProc(prog)
	require.NoError(t, err)
	proc, err := json.Marshal(parsed)
	require.NoError(t, err)
	s := api.SearchRequest{
		Space: space,
		Proc:  proc,
		Span:  nano.MaxSpan,
		Dir:   1,
	}
	// XXX Get rid of this format query param and use http headers instead.
	res := httpRequest(t, zqd.NewHandler(c), "POST", "http://localhost:9867/search?format=bzng", s)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "application/ndjson", res.Header.Get("Content-Type"))
	buf := bytes.NewBuffer(nil)
	w := zngio.NewWriter(buf)
	r := bzngio.NewReader(res.Body, resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
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

func newCore(t *testing.T) *zqd.Core {
	root := createTempDir(t)
	return newCoreAtDir(t, root)
}

func newCoreAtDir(t *testing.T, dir string) *zqd.Core {
	conf := zqd.Config{
		Root:   dir,
		Logger: zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)),
	}
	return zqd.NewCore(conf)
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

// createSpace initiates a new space in the provided zqd.Core and writes the zng
// source into said space.
func createSpaceWithData(t *testing.T, c *zqd.Core, spaceName, src string) {
	res := createSpace(t, c, spaceName, "")
	writeToSpace(t, c, res.Name, src)
}

// writeToSpace writes the provided zng source in to the provided space
// directory.
func writeToSpace(t *testing.T, c *zqd.Core, spaceName, src string) {
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
