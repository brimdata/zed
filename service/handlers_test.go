package service_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/pkg/test"
	"github.com/brimdata/zed/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var defaultLayout = order.Layout{
	Order: order.Desc,
	Keys:  field.DottedList("ts"),
}

func TestQuery(t *testing.T) {
	src := `
{_path:"b",ts:1970-01-01T00:00:01Z}
{_path:"a",ts:1970-01-01T00:00:01Z}
`
	expected := `{_path:"b",ts:1970-01-01T00:00:01Z}
`
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test", Layout: defaultLayout})
	conn.TestLoad(poolID, "main", strings.NewReader(src))
	assert.Equal(t, expected, conn.TestQuery("from test | _path == 'b'"))
}

func TestQueryEmptyPool(t *testing.T) {
	_, conn := newCore(t)
	conn.TestPoolPost(api.PoolPostRequest{Name: "test", Layout: defaultLayout})
	assert.Equal(t, "", conn.TestQuery("from test"))
}

func TestQueryGroupByReverse(t *testing.T) {
	src := `
{ts:1970-01-01T00:00:01Z,uid:"A"}
{ts:1970-01-01T00:00:01Z,uid:"B"}
{ts:1970-01-01T00:00:02Z,uid:"B"}
`
	counts := `
{ts:1970-01-01T00:00:02Z,count:1(uint64)}
{ts:1970-01-01T00:00:01Z,count:2(uint64)}
`
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test", Layout: defaultLayout})
	conn.TestLoad(poolID, "main", strings.NewReader(src))
	require.Equal(t, test.Trim(counts), conn.TestQuery("from test | count() by every(1s)"))
}

func TestPoolStats(t *testing.T) {
	src := `
{_path:"conn",ts:1970-01-01T00:00:01Z,uid:"CBrzd94qfowOqJwCHa"}
{_path:"conn",ts:1970-01-01T00:00:02Z,uid:"C8Tful1TvM3Zf5x8fl"}
`
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test", Layout: defaultLayout})
	conn.TestLoad(poolID, "main", strings.NewReader(src))

	span := nano.Span{Ts: 1e9, Dur: 1e9 + 1}
	expected := lake.PoolStats{
		Span: &span,
		Size: 84,
	}
	require.Equal(t, expected, conn.TestPoolStats(poolID))
}

func TestPoolStatsNoData(t *testing.T) {
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test", Layout: defaultLayout})
	info := conn.TestPoolStats(poolID)
	expected := lake.PoolStats{
		Size: 0,
	}
	require.Equal(t, expected, info)
}

func TestPoolPostNameOnly(t *testing.T) {
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test"})
	assert.NotEqual(t, ksuid.Nil, poolID)
}

func TestPoolPostDuplicateName(t *testing.T) {
	_, conn := newCore(t)
	conn.TestPoolPost(api.PoolPostRequest{Name: "test"})
	_, err := conn.CreatePool(context.Background(), api.PoolPostRequest{Name: "test"})
	require.Equal(t, errors.Is(err, client.ErrPoolExists), true)
}

func TestPoolInvalidName(t *testing.T) {
	t.Skip("verify invalid characters for a pool name")
	ctx := context.Background()
	_, conn := newCore(t)
	t.Run("Post", func(t *testing.T) {
		_, err := conn.CreatePool(ctx, api.PoolPostRequest{Name: "𝚭𝚴𝚪 is.good"})
		require.NoError(t, err)
		_, err = conn.CreatePool(ctx, api.PoolPostRequest{Name: "𝚭𝚴𝚪/bad"})
		require.EqualError(t, err, "status code 400: name may not contain '/' or non-printable characters")
	})
	t.Run("Put", func(t *testing.T) {
		poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "𝚭𝚴𝚪1"})
		err := conn.RenamePool(ctx, poolID, api.PoolPutRequest{Name: "𝚭𝚴𝚪/2"})
		require.EqualError(t, err, "status code 400: name may not contain '/' or non-printable characters")
	})
}

func TestPoolPutDuplicateName(t *testing.T) {
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test"})
	conn.TestPoolPost(api.PoolPostRequest{Name: "test1"})
	err := conn.RenamePool(context.Background(), poolID, api.PoolPutRequest{Name: "test"})
	assert.EqualError(t, err, "status code 409: test: pool already exists")
}

func TestPoolPut(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test"})
	err := conn.RenamePool(ctx, poolID, api.PoolPutRequest{Name: "new_name"})
	require.NoError(t, err)
	info := conn.TestPoolGet(poolID)
	assert.Equal(t, "new_name", info.Name)
}

func TestPoolRemote(t *testing.T) {
	ctx := context.Background()
	_, conn := newCore(t)
	poolID := conn.TestPoolPost(api.PoolPostRequest{Name: "test"})
	err := conn.RemovePool(ctx, poolID)
	require.NoError(t, err)
	list := conn.TestPoolList()
	require.Len(t, list, 0)
}

func TestNoEndSlashSupport(t *testing.T) {
	_, conn := newCore(t)
	_, err := conn.Do(conn.NewRequest(context.Background(), "GET", "/pool/", nil))
	require.Error(t, err)
	require.Equal(t, 404, err.(*client.ErrorResponse).StatusCode)
}

func TestRequestID(t *testing.T) {
	ctx := context.Background()
	pools := api.QueryRequest{Query: "from :pools"}
	t.Run("GeneratesUniqueID", func(t *testing.T) {
		_, conn := newCore(t)
		res1, err := conn.Do(conn.NewRequest(ctx, "POST", "/query", pools))
		require.NoError(t, err)
		res2, err := conn.Do(conn.NewRequest(ctx, "POST", "/query", pools))
		require.NoError(t, err)
		assert.NotEqual(t, "", res1.Header.Get("X-Request-ID"))
		assert.NotEqual(t, "", res2.Header.Get("X-Request-ID"))
	})
	t.Run("PropagatesID", func(t *testing.T) {
		_, conn := newCore(t)
		requestID := "random-request-ID"
		req := conn.NewRequest(context.Background(), "POST", "/query", pools)
		req.Header.Set("X-Request-ID", requestID)
		res, err := conn.Do(req)
		require.NoError(t, err)
		require.Equal(t, requestID, res.Header.Get("X-Request-ID"))
	})
}

/* Not yet
func TestIndexSearch(t *testing.T) {
	t.Skip("issue #2532")
	thresh := int64(1000)

	pool, err := conn.TestPoolPost(context.Background(), api.PoolPostRequest{
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
{key:257,count:1(uint64),first:2020-04-22T01:23:02.06699522Z,last:2020-04-22T01:13:34.06491752Z}(=0)
{key:257,count:1,first:2020-04-22T00:52:28.0632538Z,last:2020-04-22T00:43:20.06892251Z}(0)
{key:257,count:1,first:2020-04-21T23:37:25.0693411Z,last:2020-04-21T23:28:29.06845389Z}(0)
{key:257,count:1,first:2020-04-21T23:28:23.06774599Z,last:2020-04-21T23:19:42.064686Z}(0)
{key:257,count:1,first:2020-04-21T23:11:06.06396109Z,last:2020-04-21T23:01:02.069881Z}(0)
{key:257,count:1,first:2020-04-21T22:51:17.06450528Z,last:2020-04-21T22:40:30.06852324Z}(0)
`
	res, _ := indexSearch(t, conn, pool.ID, "", []string{"v=257"})
	assert.Equal(t, test.Trim(exp), zsonCopy(t, "drop _log", res))
}

func indexSearch(t *testing.T, conn *testClient, pool ksuid.KSUID, indexName string, patterns []string) (string, []interface{}) {
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
*/
func newCore(t *testing.T) (*service.Core, *testClient) {
	root := t.TempDir()
	return newCoreAtDir(t, root)
}

func newCoreAtDir(t *testing.T, dir string) (*service.Core, *testClient) {
	t.Cleanup(func() { os.RemoveAll(dir) })
	return newCoreWithConfig(t, service.Config{Root: storage.MustParseURI(dir)})
}

func newCoreWithConfig(t *testing.T, conf service.Config) (*service.Core, *testClient) {
	if conf.Root == nil {
		conf.Root = storage.MustParseURI(t.TempDir())
	}
	if conf.Logger == nil {
		conf.Logger = zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel))
	}
	core, err := service.NewCore(context.Background(), conf)
	require.NoError(t, err)
	srv := httptest.NewServer(core)
	t.Cleanup(srv.Close)
	return core, &testClient{
		Connection: client.NewConnectionTo(srv.URL),
		T:          t,
	}
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
