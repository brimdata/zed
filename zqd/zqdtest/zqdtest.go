package zqdtest

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

// CreateSpace creates a new temp dir to house a space and writes the provided
// zng source into said space.
func CreateSpace(t *testing.T, src string) string {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	WriteToSpace(t, dir, src)
	return dir
}

// WriteToSpace writes the provided zng source in to the provided space
// directory.
func WriteToSpace(t *testing.T, space, src string) {
	f, err := os.Create(filepath.Join(space, "all.bzng"))
	require.NoError(t, err)
	defer f.Close()
	// write zng
	w := bzngio.NewWriter(f)
	r := zngio.NewReader(strings.NewReader(src), resolver.NewContext())
	require.NoError(t, zbuf.Copy(zbuf.NopFlusher(w), r))
}

func NewRequest(method, url string, body interface{}) *http.Request {
	var buf *bytes.Buffer
	if body != nil {
		buf = bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			panic(err)
		}
	}
	return httptest.NewRequest(method, url, buf)
}

func ExecRequest(h http.Handler, r *http.Request) *http.Response {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Result()
}
