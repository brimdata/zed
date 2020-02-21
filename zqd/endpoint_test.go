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
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/require"
)

func TestSimpleSearch(t *testing.T) {
	src := `#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]
`
	space := createSpace(t, src)
	defer os.RemoveAll(space)
	require.Equal(t, src, execSearch(t, space, "*"))
}

func TestSpaceInfo(t *testing.T) {
	space := createSpace(t, `#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]`)
	defer os.RemoveAll(space)
	min := nano.Unix(1521911721, 255387000)
	max := nano.Unix(1521911723, 205187000)
	expected := api.SpaceInfo{
		Name:          space,
		MinTime:       &min,
		MaxTime:       &max,
		Size:          88,
		PacketSupport: false,
	}
	u := fmt.Sprintf("http://localhost:9867/space/%s/", url.PathEscape(space))
	res := execRequest(zqd.NewHandler(), "GET", u, nil)
	require.Equal(t, http.StatusOK, res.StatusCode)
	var info api.SpaceInfo
	err := json.NewDecoder(res.Body).Decode(&info)
	require.NoError(t, err)
	require.Equal(t, expected, info)
}

func execSearch(t *testing.T, space, prog string) string {
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
	res := execRequest(zqd.NewHandler(), "POST", "http://localhost:9867/search/?format=bzng", s)
	require.Equal(t, http.StatusOK, res.StatusCode)
	buf := bytes.NewBuffer(nil)
	w := zngio.NewWriter(buf)
	r := bzngio.NewReader(res.Body, resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
	return buf.String()
}

// createSpace creates a new temp dir to house a space and writes the provided
// zng source into said space.
func createSpace(t *testing.T, src string) string {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	writeToSpace(t, dir, src)
	return dir
}

// writeToSpace writes the provided zng source in to the provided space
// directory.
func writeToSpace(t *testing.T, space, src string) {
	f, err := os.Create(filepath.Join(space, "all.bzng"))
	require.NoError(t, err)
	defer f.Close()
	// write zng
	w := bzngio.NewWriter(f)
	r := zngio.NewReader(strings.NewReader(src), resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
}

func execRequest(h http.Handler, method, url string, body interface{}) *http.Response {
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
