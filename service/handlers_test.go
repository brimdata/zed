package service_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/test"
	"github.com/brimdata/zed/service"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	babble       = "../../testdata/babble.zson"
	babbleSorted = "../../testdata/babble-sorted.zson"
)

func TestASTPost(t *testing.T) {
	_, conn := newCore(t)
	resp, err := conn.Do(context.Background(), http.MethodPost, "/ast", &api.ASTRequest{ZQL: "*"})
	require.NoError(t, err)
	require.Equal(t, string(resp.Body()), "{\"kind\":\"Sequential\",\"procs\":[{\"kind\":\"Filter\",\"expr\":{\"kind\":\"Primitive\",\"type\":\"bool\",\"text\":\"true\"}}]}\n")
}

func TestSearch(t *testing.T) {
	const src = `
{_path:"conn",ts:2018-03-24T17:15:23.205187Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
{_path:"conn",ts:2018-03-24T17:15:21.255387Z,uid:"C8Tful1TvM3Zf5x8fl"} (0)
`
	_, conn := newCore(t)
	ctx := context.Background()
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test", Order: order.Desc})
	require.NoError(t, err)
	_, err = conn.LogPostReaders(context.Background(), pool.ID, nil, strings.NewReader(src))
	require.NoError(t, err)

	res := searchZson(t, conn, pool.ID, "*")
	require.Equal(t, test.Trim(src), res)
}

func TestSearchNoCtrl(t *testing.T) {
	src := `
{_path:"conn",ts:2018-03-24T17:15:23.205187Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
{_path:"conn",ts:2018-03-24T17:15:21.255387Z,uid:"C8Tful1TvM3Zf5x8fl"} (0)
`
	_, conn := newCore(t)
	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test", Order: order.Desc})
	require.NoError(t, err)
	_, err = conn.LogPostReaders(context.Background(), pool.ID, nil, strings.NewReader(src))
	require.NoError(t, err)

	parsed, err := compiler.ParseProc("*")
	require.NoError(t, err)
	proc, err := json.Marshal(parsed)
	require.NoError(t, err)
	req := api.SearchRequest{
		Pool: pool.ID,
		Proc: proc,
		Span: nano.MaxSpan,
		Dir:  -1,
	}
	body, err := conn.SearchRaw(context.Background(), req, map[string]string{"noctrl": "true"})
	require.NoError(t, err)
	var msgs []interface{}
	r := client.NewZngSearch(body)
	r.SetOnCtrl(func(i interface{}) {
		msgs = append(msgs, i)
	})
	buf := bytes.NewBuffer(nil)
	w := zsonio.NewWriter(zio.NopCloser(buf), zsonio.WriterOpts{})
	require.NoError(t, zio.Copy(w, r))
	require.Equal(t, test.Trim(src), buf.String())
	require.Equal(t, 0, len(msgs))
}

func TestSearchStats(t *testing.T) {
	src := `
{_path:"a",ts:1970-01-01T00:00:01Z}
{_path:"b",ts:1970-01-01T00:00:01Z}
`
	_, conn := newCore(t)
	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	_, err = conn.LogPostReaders(context.Background(), pool.ID, nil, strings.NewReader(src))
	require.NoError(t, err)
	_, msgs := search(t, conn, pool.ID, "_path != b")
	var stats *api.SearchStats
	for i := len(msgs) - 1; i >= 0; i-- {
		if s, ok := msgs[i].(*api.SearchStats); ok {
			stats = s
			break
		}
	}
	require.NotNil(t, stats)
	assert.Equal(t, stats.Type, "SearchStats")
	assert.Equal(t, stats.ScannerStats, api.ScannerStats{
		BytesRead:      14,
		BytesMatched:   7,
		RecordsRead:    2,
		RecordsMatched: 1,
	})
}

func TestGroupByReverse(t *testing.T) {
	src := `
{ts:1970-01-01T00:00:01Z,uid:"A"} (=0)
{ts:1970-01-01T00:00:01Z,uid:"B"} (0)
{ts:1970-01-01T00:00:02Z,uid:"B"} (0)
`
	counts := `
{ts:1970-01-01T00:00:02Z,count:1 (uint64)} (=0)
{ts:1970-01-01T00:00:01Z,count:2} (0)
`
	_, conn := newCore(t)
	keys := field.List{field.New("ts")}
	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test", Keys: keys, Order: order.Desc})
	require.NoError(t, err)
	_, err = conn.LogPostReaders(context.Background(), pool.ID, nil, strings.NewReader(src))
	require.NoError(t, err)
	res := searchZson(t, conn, pool.ID, "every 1s count()")
	require.Equal(t, test.Trim(counts), res)
}

func TestSearchEmptyPool(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	res, err := conn.Search(context.Background(), pool.ID, "*")
	require.NoError(t, err)
	w := zsonio.NewWriter(zio.NopCloser(io.Discard), zsonio.WriterOpts{})
	err = zio.Copy(w, res)
	assert.NoError(t, err, nil)
}

func TestSearchError(t *testing.T) {
	src := `
{_path:"conn",ts:2018-03-24T17:15:23.205187Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
{_path:"conn",ts:2018-03-24T17:15:21.255387Z,uid:"C8Tful1TvM3Zf5x8fl"} (0)
`
	_, conn := newCore(t)
	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	_, err = conn.LogPostReaders(context.Background(), pool.ID, nil, strings.NewReader(src))
	require.NoError(t, err)

	parsed, err := compiler.ParseProc("*")
	require.NoError(t, err)
	proc, err := json.Marshal(parsed)
	require.NoError(t, err)
	t.Run("InvalidDir", func(t *testing.T) {
		req := api.SearchRequest{
			Pool: pool.ID,
			Proc: proc,
			Span: nano.MaxSpan,
			Dir:  2,
		}
		_, err = conn.SearchRaw(context.Background(), req, nil)
		require.Error(t, err)
		errResp := err.(*client.ErrorResponse)
		assert.Equal(t, http.StatusBadRequest, errResp.StatusCode())
		assert.IsType(t, &api.Error{}, errResp.Err)
	})
	t.Run("ForwardSearchUnsupported", func(t *testing.T) {
		req := api.SearchRequest{
			Pool: pool.ID,
			Proc: proc,
			Span: nano.MaxSpan,
			Dir:  1,
		}
		_, err = conn.SearchRaw(context.Background(), req, nil)
		require.Error(t, err)
		errResp := err.(*client.ErrorResponse)
		assert.Equal(t, http.StatusBadRequest, errResp.StatusCode())
		assert.IsType(t, &api.Error{}, errResp.Err)
	})
}

func TestPoolInfo(t *testing.T) {
	src := `
{_path:"conn",ts:1970-01-01T00:00:01Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
{_path:"conn",ts:1970-01-01T00:00:02Z,uid:"C8Tful1TvM3Zf5x8fl"} (0)
`
	ctx := context.Background()
	_, conn := newCore(t)
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test", Order: order.Desc})
	require.NoError(t, err)
	_, err = conn.LogPostReaders(context.Background(), pool.ID, nil, strings.NewReader(src))
	require.NoError(t, err)

	span := nano.Span{Ts: 1e9, Dur: 1e9 + 1}
	expected := &api.PoolInfo{
		Pool: api.Pool{
			ID:   pool.ID,
			Name: pool.Name,
		},
		Span: &span,
		Size: 58,
	}
	info, err := conn.PoolInfo(ctx, pool.ID)
	require.NoError(t, err)
	require.Equal(t, expected, info)
}

func TestPoolInfoNoData(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	info, err := conn.PoolInfo(ctx, pool.ID)
	require.NoError(t, err)
	expected := &api.PoolInfo{
		Pool: api.Pool{
			ID:   pool.ID,
			Name: pool.Name,
		},
		Size: 0,
	}
	require.Equal(t, expected, info)
}

func TestPoolPostNameOnly(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	assert.Equal(t, "test", pool.Name)
	assert.NotEqual(t, "", pool.ID)
}

func TestPoolPostDuplicateName(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	_, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	_, err = conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.Equal(t, client.ErrPoolExists, err)
}

func TestPoolInvalidName(t *testing.T) {
	t.Skip("verify invalid characters for a pool name")
	ctx := context.Background()
	_, conn := newCore(t)
	t.Run("Post", func(t *testing.T) {
		_, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "𝚭𝚴𝚪 is.good"})
		require.NoError(t, err)
		_, err = conn.PoolPost(ctx, api.PoolPostRequest{Name: "𝚭𝚴𝚪/bad"})
		require.EqualError(t, err, "status code 400: name may not contain '/' or non-printable characters")
	})
	t.Run("Put", func(t *testing.T) {
		pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "𝚭𝚴𝚪1"})
		require.NoError(t, err)
		err = conn.PoolPut(ctx, pool.ID, api.PoolPutRequest{Name: "𝚭𝚴𝚪/2"})
		require.EqualError(t, err, "status code 400: name may not contain '/' or non-printable characters")
	})
}

func TestPoolPutDuplicateName(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	_, err = conn.PoolPost(ctx, api.PoolPostRequest{Name: "test1"})
	require.NoError(t, err)
	err = conn.PoolPut(ctx, pool.ID, api.PoolPutRequest{Name: "test"})
	assert.EqualError(t, err, "status code 409: pool already exists")
}

func TestPoolPut(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	err = conn.PoolPut(ctx, pool.ID, api.PoolPutRequest{Name: "new_name"})
	require.NoError(t, err)
	info, err := conn.PoolInfo(ctx, pool.ID)
	assert.Equal(t, "new_name", info.Name)
}

func TestPoolDelete(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	pool, err := conn.PoolPost(ctx, api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)
	err = conn.PoolDelete(ctx, pool.ID)
	require.NoError(t, err)
	list, err := conn.PoolList(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestNoEndSlashSupport(t *testing.T) {
	_, conn := newCore(t)
	_, err := conn.Do(context.Background(), "GET", "/pool/", nil)
	require.Error(t, err)
	require.Equal(t, 404, err.(*client.ErrorResponse).StatusCode())
}

func TestRequestID(t *testing.T) {
	ctx := context.Background()
	t.Run("GeneratesUniqueID", func(t *testing.T) {
		_, conn := newCore(t)
		res1, err := conn.Do(ctx, "GET", "/pool", nil)
		require.NoError(t, err)
		res2, err := conn.Do(ctx, "GET", "/pool", nil)
		require.NoError(t, err)
		assert.NotEqual(t, "", res1.Header().Get("X-Request-ID"))
		assert.NotEqual(t, "", res2.Header().Get("X-Request-ID"))
	})
	t.Run("PropagatesID", func(t *testing.T) {
		_, conn := newCore(t)
		requestID := "random-request-ID"
		req := conn.Request(context.Background())
		req.SetHeader("X-Request-ID", requestID)
		res, err := req.Execute("GET", "/pool")
		require.NoError(t, err)
		require.Equal(t, requestID, res.Header().Get("X-Request-ID"))
	})
}

func TestPostZsonLogs(t *testing.T) {
	const src1 = `
{_path:"conn",ts:1970-01-01T00:00:01Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
`
	const src2 = `
{_path:"conn",ts:1970-01-01T00:00:02Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
`
	const expected = `
{_path:"conn",ts:1970-01-01T00:00:02Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
{_path:"conn",ts:1970-01-01T00:00:01Z,uid:"CBrzd94qfowOqJwCHa"} (0)
`

	_, conn := newCore(t)
	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test", Order: order.Desc})
	require.NoError(t, err)

	postResp, err := conn.LogPostReaders(context.Background(), pool.ID, nil,
		strings.NewReader(src1),
		strings.NewReader(src2),
	)
	require.NoError(t, err)
	assert.Equal(t, "LogPostResponse", postResp.Type)
	assert.EqualValues(t, 160, postResp.BytesRead)

	res := searchZson(t, conn, pool.ID, "*")
	require.EqualValues(t, test.Trim(expected), res)

	info, err := conn.PoolInfo(context.Background(), pool.ID)
	require.NoError(t, err)
	assert.Equal(t, &api.PoolInfo{
		Pool: api.Pool{
			ID:   pool.ID,
			Name: pool.Name,
		},
		Span: &nano.Span{Ts: nano.Ts(nano.Second), Dur: nano.Second + 1},
		Size: 58,
	}, info)
}

func TestPostZngLogWarning(t *testing.T) {
	const src1 = `undetectableformat`
	const src2 = `
{_path:"conn",ts:1970-01-01T00:00:01Z,uid:"CBrzd94qfowOqJwCHa" (bstring)} (=0)
detectablebutbadline`

	_, conn := newCore(t)
	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)

	res, err := conn.LogPostReaders(context.Background(), pool.ID, nil,
		strings.NewReader(src1),
		strings.NewReader(src2),
	)
	require.NoError(t, err)
	assert.Regexp(t, ": format detection error.*", res.Warnings[0])
	assert.Exactly(t, `data2: identifier "detectablebutbadline" must be enum and requires decorator`, res.Warnings[1])
}

func TestPostNDJSONLogs(t *testing.T) {
	const src = `{"ts":"1000","uid":"CXY9a54W2dLZwzPXf1","_path":"http"}
{"ts":"2000","uid":"CXY9a54W2dLZwzPXf1","_path":"http"}`
	const expected = `{ts:"2000",uid:"CXY9a54W2dLZwzPXf1",_path:"http"}
{ts:"1000",uid:"CXY9a54W2dLZwzPXf1",_path:"http"}`

	test := func(input string) {
		_, conn := newCore(t)

		pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test", Order: order.Desc})
		require.NoError(t, err)

		_, err = conn.LogPostReaders(context.Background(), pool.ID, nil, strings.NewReader(src))
		require.NoError(t, err)

		res := searchZson(t, conn, pool.ID, "*")
		require.Equal(t, expected, strings.TrimSpace(res))

		info, err := conn.PoolInfo(context.Background(), pool.ID)
		require.NoError(t, err)
		span := nano.Span{Ts: 0, Dur: 1}
		require.Equal(t, &api.PoolInfo{
			Pool: api.Pool{
				ID:   pool.ID,
				Name: pool.Name,
			},
			Size: 58,
			Span: &span,
		}, info)
	}
	t.Run("plain", func(t *testing.T) {
		test(src)
	})
	t.Run("gzipped", func(t *testing.T) {
		var b strings.Builder
		w := gzip.NewWriter(&b)
		_, err := w.Write([]byte(src))
		require.NoError(t, err)
		require.NoError(t, w.Close())
		test(b.String())
	})
}

// Other attempts at malformed ZSON closer to the original are not yet flagged
// as errors. See https://github.com/brimdata/zed/issues/2057#issuecomment-803148819
func TestPostLogStopErr(t *testing.T) {
	const src = `
{_path:"conn",ts:1970-01-01T00:00:01Z,uid:"CBrzd94qfowOqJwCHa" (bstring} (=0)
`

	_, conn := newCore(t)
	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{Name: "test"})
	require.NoError(t, err)

	opts := &client.LogPostOpts{StopError: true}
	_, err = conn.LogPostReaders(context.Background(), pool.ID, opts, strings.NewReader(src))
	require.Error(t, err)
	assert.Regexp(t, ": format detection error.*", err.Error())
}

func TestIndexSearch(t *testing.T) {
	t.Skip("issue #2532")
	thresh := int64(1000)
	root := t.TempDir()

	_, conn := newCoreAtDir(t, root)

	pool, err := conn.PoolPost(context.Background(), api.PoolPostRequest{
		Name:   "TestIndexSearch",
		Thresh: thresh,
	})
	require.NoError(t, err)
	// babbleSorted must be used because regular babble isn't fully sorted and
	// generates an overlap which on compaction deletes certain indices. We
	// should be able to remove this once #1656 is completed and we have some
	// api way of determining if compactions are complete.
	_, err = conn.LogPost(context.Background(), pool.ID, nil, babbleSorted)
	require.NoError(t, err)
	err = conn.IndexPost(context.Background(), pool.ID, api.IndexPostRequest{
		Patterns: []string{"v"},
	})
	require.NoError(t, err)

	exp := `
{key:257,count:1 (uint64),first:2020-04-22T01:23:02.06699522Z,last:2020-04-22T01:13:34.06491752Z} (=0)
{key:257,count:1,first:2020-04-22T00:52:28.0632538Z,last:2020-04-22T00:43:20.06892251Z} (0)
{key:257,count:1,first:2020-04-21T23:37:25.0693411Z,last:2020-04-21T23:28:29.06845389Z} (0)
{key:257,count:1,first:2020-04-21T23:28:23.06774599Z,last:2020-04-21T23:19:42.064686Z} (0)
{key:257,count:1,first:2020-04-21T23:11:06.06396109Z,last:2020-04-21T23:01:02.069881Z} (0)
{key:257,count:1,first:2020-04-21T22:51:17.06450528Z,last:2020-04-21T22:40:30.06852324Z} (0)
`
	res, _ := indexSearch(t, conn, pool.ID, "", []string{"v=257"})
	assert.Equal(t, test.Trim(exp), zsonCopy(t, "drop _log", res))
}

func indexSearch(t *testing.T, conn *client.Connection, pool ksuid.KSUID, indexName string, patterns []string) (string, []interface{}) {
	req := api.IndexSearchRequest{
		IndexName: indexName,
		Patterns:  patterns,
	}
	r, err := conn.IndexSearch(context.Background(), pool, req, nil)
	require.NoError(t, err)
	buf := bytes.NewBuffer(nil)
	w := zsonio.NewWriter(zio.NopCloser(buf), zsonio.WriterOpts{})
	var msgs []interface{}
	r.SetOnCtrl(func(i interface{}) {
		msgs = append(msgs, i)
	})
	require.NoError(t, zio.Copy(w, r))
	return buf.String(), msgs
}

// search runs the provided zql program as a search on the provided
// pool, returning the zson results along with a slice of all control
// messages that were received.
func search(t *testing.T, conn *client.Connection, pool ksuid.KSUID, prog string) (string, []interface{}) {
	parsed, err := compiler.ParseProc(prog)
	require.NoError(t, err)
	proc, err := json.Marshal(parsed)
	require.NoError(t, err)
	req := api.SearchRequest{
		Pool: pool,
		Proc: proc,
		Span: nano.MaxSpan,
		Dir:  -1,
	}
	body, err := conn.SearchRaw(context.Background(), req, nil)
	require.NoError(t, err)
	r := client.NewZngSearch(body)
	buf := bytes.NewBuffer(nil)
	w := zsonio.NewWriter(zio.NopCloser(buf), zsonio.WriterOpts{})
	var msgs []interface{}
	r.SetOnCtrl(func(i interface{}) {
		msgs = append(msgs, i)
	})
	require.NoError(t, zio.Copy(w, r))
	return buf.String(), msgs
}

func searchZson(t *testing.T, conn *client.Connection, pool ksuid.KSUID, prog string) string {
	res, err := conn.Search(context.Background(), pool, prog)
	require.NoError(t, err)
	buf := bytes.NewBuffer(nil)
	w := zsonio.NewWriter(zio.NopCloser(buf), zsonio.WriterOpts{})
	err = zio.Copy(w, res)
	require.NoError(t, err)
	return buf.String()
}

func zsonCopy(t *testing.T, prog string, in string) string {
	zctx := zson.NewContext()
	r := zson.NewReader(strings.NewReader(in), zctx)
	var buf bytes.Buffer
	w := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	p := compiler.MustParseProc(prog)
	err := driver.Copy(context.Background(), w, p, zctx, r, nil)
	require.NoError(t, err)
	return buf.String()
}

func writeTempFile(t *testing.T, data string) string {
	f, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)
	name := f.Name()
	t.Cleanup(func() { os.Remove(name) })
	_, err = f.WriteString(data)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return name
}

func newCore(t *testing.T) (*service.Core, *client.Connection) {
	root := t.TempDir()
	return newCoreAtDir(t, root)
}

func newCoreAtDir(t *testing.T, dir string) (*service.Core, *client.Connection) {
	t.Cleanup(func() { os.RemoveAll(dir) })
	return newCoreWithConfig(t, service.Config{Root: dir})
}

func newCoreWithConfig(t *testing.T, conf service.Config) (*service.Core, *client.Connection) {
	if conf.Root == "" {
		conf.Root = t.TempDir()
	}
	if conf.Logger == nil {
		conf.Logger = zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	}
	core, err := service.NewCore(context.Background(), conf)
	require.NoError(t, err)
	srv := httptest.NewServer(core)
	t.Cleanup(srv.Close)
	return core, client.NewConnectionTo(srv.URL)
}

func promCounterValue(g prometheus.Gatherer, name string) interface{} {
	metricFamilies, err := g.Gather()
	if err != nil {
		return err
	}
	for _, mf := range metricFamilies {
		if mf.GetName() == name {
			return mf.GetMetric()[0].GetCounter().GetValue()
		}
	}
	return errors.New("metric not found")
}
