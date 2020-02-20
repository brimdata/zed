package space_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqd/zqdtest"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestSpaceInfo(t *testing.T) {
	space := zqdtest.CreateSpace(t, `#0:record[_path:string,ts:time,uid:bstring]
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
	res := zqdtest.ExecRequest(newRouter(), zqdtest.NewRequest("GET", u, nil))
	require.Equal(t, http.StatusOK, res.StatusCode)
	var info api.SpaceInfo
	err := json.NewDecoder(res.Body).Decode(&info)
	require.NoError(t, err)
	require.Equal(t, expected, info)
}

func newRouter() *mux.Router {
	router := mux.NewRouter()
	space.AddRoutes(router)
	return router
}
