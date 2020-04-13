package ingest

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/space"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTempSpace(t *testing.T) (string, *space.Space) {
	root, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	s, err := space.Create(root, t.Name(), "")
	require.NoError(t, err)
	return root, s
}
func writeTempFile(t *testing.T, data string) string {
	f, err := ioutil.TempFile("", "testfile")
	require.NoError(t, err)
	name := f.Name()
	defer f.Close()
	_, err = f.WriteString(data)
	require.NoError(t, err)
	return name
}

func TestLogsErrInFlight(t *testing.T) {
	src := `
#0:record[_path:string,ts:time,uid:bstring]
0:[conn;1521911723.205187;CBrzd94qfowOqJwCHa;]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;]
`
	root, s := createTempSpace(t)
	defer os.RemoveAll(root)
	f := writeTempFile(t, src)

	errCh1 := make(chan error)
	errCh2 := make(chan error)
	go func() {
		// xxx this test is now a false positive. to be updated before merging.
		p := api.NewJSONPipe(httptest.NewRecorder())
		errCh1 <- Logs(context.Background(), p, s, []string{f}, nil, 10)
	}()
	go func() {
		p := api.NewJSONPipe(httptest.NewRecorder())
		errCh2 <- Logs(context.Background(), p, s, []string{f}, nil, 10)
	}()
	err1 := <-errCh1
	err2 := <-errCh2
	if err1 == nil {
		assert.EqualError(t, err2, ErrIngestProcessInFlight.Error())
		return
	}
	if err2 == nil {
		assert.EqualError(t, err1, ErrIngestProcessInFlight.Error())
		return
	}
	assert.Fail(t, "expected only one error")
}
