package zqd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/stretchr/testify/require"
)

func TestSimpleSearch(t *testing.T) {
	src := `
#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]
`
	space := "test"
	root := createTempDir(t)
	defer os.RemoveAll(root)
	createSpaceWithData(t, root, space, src)
	require.Equal(t, test.Trim(src), execSearch(t, root, space, "*"))
}

func TestSpaceList(t *testing.T) {
	root := createTempDir(t)
	defer os.RemoveAll(root)
	sp1 := createSpace(t, root, "sp1", "")
	sp2 := createSpace(t, root, "sp2", "")
	sp3 := createSpace(t, root, "sp3", "")
	sp4 := createSpace(t, root, "sp4", "")
	// delete config.json from sp3
	require.NoError(t, os.Remove(filepath.Join(root, sp3.Name, "config.json")))
	expected := []string{
		sp1.Name,
		sp2.Name,
		sp4.Name,
	}
	body := httpSuccess(t, zqd.NewHandler(root), "GET", "http://localhost:9867/space", nil)
	var list []string
	err := json.NewDecoder(body).Decode(&list)
	require.NoError(t, err)
	require.Equal(t, expected, list)
}

func TestSpaceInfo(t *testing.T) {
	space := "test"
	src := `
#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]`
	root := createTempDir(t)
	defer os.RemoveAll(root)
	createSpaceWithData(t, root, space, src)
	min := nano.Unix(1521911721, 255387000)
	max := nano.Unix(1521911723, 205187000)
	expected := api.SpaceInfo{
		Name:          space,
		MinTime:       &min,
		MaxTime:       &max,
		Size:          88,
		PacketSupport: false,
	}
	u := fmt.Sprintf("http://localhost:9867/space/%s", space)
	body := httpSuccess(t, zqd.NewHandler(root), "GET", u, nil)
	var info api.SpaceInfo
	err := json.NewDecoder(body).Decode(&info)
	require.NoError(t, err)
	require.Equal(t, expected, info)
}

func TestSpacePostNameOnly(t *testing.T) {
	root := createTempDir(t)
	defer os.RemoveAll(root)
	expected := api.SpacePostResponse{
		Name:    "test",
		DataDir: filepath.Join(root, "test"),
	}
	res := createSpace(t, root, "test", "")
	require.Equal(t, expected, res)
}

func TestSpacePostDataDirOnly(t *testing.T) {
	run := func(name string, cb func(t *testing.T, tmp, root string) (string, api.SpacePostResponse)) {
		tmp := createTempDir(t)
		defer os.RemoveAll(tmp)
		root := filepath.Join(tmp, "spaces")
		require.NoError(t, os.Mkdir(root, 0755))
		t.Run(name, func(t *testing.T) {
			datadir, expected := cb(t, tmp, root)
			res := createSpace(t, root, "", datadir)
			require.Equal(t, expected, res)
		})
	}
	run("Simple", func(t *testing.T, tmp, root string) (string, api.SpacePostResponse) {
		datadir := filepath.Join(tmp, "mypcap.brim")
		require.NoError(t, os.Mkdir(datadir, 0755))
		return datadir, api.SpacePostResponse{
			Name:    "mypcap.brim",
			DataDir: datadir,
		}
	})
	run("DuplicateName", func(t *testing.T, tmp, root string) (string, api.SpacePostResponse) {
		createSpace(t, root, "mypcap.brim", "")
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
	run := func(name string, cb func(t *testing.T, tmp, root string)) {
		tmp := createTempDir(t)
		defer os.RemoveAll(tmp)
		root := filepath.Join(tmp, "spaces")
		require.NoError(t, os.Mkdir(root, 0755))
		t.Run(name, func(t *testing.T) {
			cb(t, tmp, root)
			// make sure no spaces exist
			r := httpSuccess(t, zqd.NewHandler(root), "GET", "http://localhost:9867/space", nil)
			body, err := ioutil.ReadAll(r)
			require.NoError(t, err)
			require.Equal(t, "[]\n", string(body))
		})
	}
	run("Simple", func(t *testing.T, tmp, root string) {
		createSpace(t, root, space, "")
		httpSuccess(t, zqd.NewHandler(root), "DELETE", spaceUrl, nil)
	})
	run("DeletesOutsideDataDir", func(t *testing.T, tmp, root string) {
		datadir := filepath.Join(tmp, "datadir")
		createSpace(t, root, space, datadir)
		httpSuccess(t, zqd.NewHandler(root), "DELETE", spaceUrl, nil)
		_, err := os.Stat(datadir)
		require.Error(t, err)
		require.Truef(t, os.IsNotExist(err), "expected error to be os.IsNotExist, got %v", err)
	})
}

func execSearch(t *testing.T, root, space, prog string) string {
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
	body := httpSuccess(t, zqd.NewHandler(root), "POST", "http://localhost:9867/search?format=bzng", s)
	buf := bytes.NewBuffer(nil)
	w := zngio.NewWriter(buf)
	r := bzngio.NewReader(body, resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
	return buf.String()
}

func createTempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", t.Name())
	require.NoError(t, err)
	return dir
}

func createSpace(t *testing.T, root, spaceName, datadir string) api.SpacePostResponse {
	req := api.SpacePostRequest{
		Name:    spaceName,
		DataDir: datadir,
	}
	body := httpSuccess(t, zqd.NewHandler(root), "POST", "http://localhost:9867/space", req)
	var res api.SpacePostResponse
	require.NoError(t, json.NewDecoder(body).Decode(&res))
	return res
}

// createSpace initiates a new space in the provided root and writes the zng
// source into said space.
func createSpaceWithData(t *testing.T, root, spaceName, src string) {
	res := createSpace(t, root, spaceName, "")
	writeToSpace(t, root, res.Name, src)
}

// writeToSpace writes the provided zng source in to the provided space
// directory.
func writeToSpace(t *testing.T, root, spaceName, src string) {
	s, err := space.Open(root, spaceName)
	require.NoError(t, err)
	f, err := s.CreateFile("all.bzng")
	require.NoError(t, err)
	defer f.Close()
	// write zng
	w := bzngio.NewWriter(f)
	r := zngio.NewReader(strings.NewReader(src), resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
}

func httpSuccess(t *testing.T, h http.Handler, method, url string, body interface{}) io.ReadCloser {
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
	res := w.Result()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(res.Body)
		require.Equal(t, http.StatusOK, res.StatusCode, string(body))
	}
	return res.Body
}

func TestNoEndSlashSupport(t *testing.T) {
	root := createTempDir(t)
	defer os.RemoveAll(root)

	h := zqd.NewHandler(root)
	r := httptest.NewRequest("GET", "http://localhost:9867/space/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	res := w.Result()
	require.Equal(t, 404, res.StatusCode)
}
