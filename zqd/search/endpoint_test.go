package search_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/zqdtest"
	"github.com/brimsec/zq/zql"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestSimpleSearch(t *testing.T) {
	src := `#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]
`
	space := zqdtest.CreateSpace(t, src)
	defer os.RemoveAll(space)
	require.Equal(t, src, exec(t, space, "*"))
}

func exec(t *testing.T, space, prog string) string {
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
	req := zqdtest.NewRequest("POST", "http://localhost:9867/search/?format=bzng", s)
	res := zqdtest.ExecRequest(newRouter(), req)
	require.Equal(t, http.StatusOK, res.StatusCode)
	buf := bytes.NewBuffer(nil)
	w := zngio.NewWriter(buf)
	r := bzngio.NewReader(res.Body, resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
	return buf.String()
}

func newRouter() *mux.Router {
	router := mux.NewRouter()
	search.AddRoutes(router)
	return router
}
