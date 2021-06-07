package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	*testing.T
	*client.Connection
}

func (c *testClient) unmarshal(r *client.ReadCloser, i interface{}) {
	zr := c.zioreader(r)
	var buf bytes.Buffer
	zw := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	require.NoError(c, zio.Copy(zw, zr))
	require.NoError(c, zw.Close())
	if s := buf.String(); s != "" {
		require.NoError(c, zson.Unmarshal(s, i))
	}
}

func (c *testClient) TestPoolStats(id ksuid.KSUID) (info lake.PoolStats) {
	r, err := c.Connection.PoolStats(context.Background(), id)
	require.NoError(c, err)
	c.unmarshal(r, &info)
	return info
}

func (c *testClient) TestPoolGet(id ksuid.KSUID) (config lake.PoolConfig) {
	r, err := c.Connection.PoolGet(context.Background(), id)
	require.NoError(c, err)
	c.unmarshal(r, &config)
	return config
}

func (c *testClient) zioreader(rc *client.ReadCloser) zio.Reader {
	format, err := api.MediaTypeToFormat(rc.ContentType)
	require.NoError(c, err)
	zr, err := anyio.NewReaderWithOpts(rc, zson.NewContext(), anyio.ReaderOpts{Format: format})
	require.NoError(c, err)
	return zr
}

func (c *testClient) TestPoolList() []lake.PoolConfig {
	r, err := c.ScanPools(context.Background())
	require.NoError(c, err)
	var confs []lake.PoolConfig
	zr := c.zioreader(r)
	for {
		rec, err := zr.Read()
		require.NoError(c, err)
		if rec == nil {
			return confs
		}
		var pool lake.PoolConfig
		err = zson.UnmarshalZNGRecord(rec, &pool)
		require.NoError(c, err)
		confs = append(confs, pool)
	}
}

func (c *testClient) TestPoolPost(payload api.PoolPostRequest) lake.PoolConfig {
	r, err := c.Connection.PoolPost(context.Background(), payload)
	require.NoError(c, err)
	var conf lake.PoolConfig
	c.unmarshal(r, &conf)
	return conf
}

func (c *testClient) TestQuery(query string) string {
	r, err := c.Connection.Query(context.Background(), query)
	require.NoError(c, err)
	zr := c.zioreader(r)
	var buf bytes.Buffer
	zw := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	require.NoError(c, zio.Copy(zw, zr))
	return buf.String()
}

func (c *testClient) TestLogPostReaders(id ksuid.KSUID, opts *client.LogPostOpts, readers ...io.Reader) (res api.LogPostResponse) {
	r, err := c.Connection.LogPostReaders(context.Background(), storage.NewLocalEngine(), id, opts, readers...)
	require.NoError(c, err)
	c.unmarshal(r, &res)
	return res
}

func (c *testClient) TestLoad(id ksuid.KSUID, r io.Reader) {
	rc, err := c.Connection.Add(context.Background(), id, r)
	require.NoError(c, err)
	var add api.AddResponse
	c.unmarshal(rc, &add)
	err = c.Connection.Commit(context.Background(), id, add.Commit, api.CommitRequest{})
	require.NoError(c, err)
}

func (c *testClient) TestAuthMethod() (res api.AuthMethodResponse) {
	r, err := c.Connection.AuthMethod(context.Background())
	require.NoError(c, err)
	c.unmarshal(r, &res)
	return res
}

func (c *testClient) TestAuthIdentity() (res api.AuthIdentityResponse) {
	r, err := c.Connection.AuthIdentity(context.Background())
	require.NoError(c, err)
	c.unmarshal(r, &res)
	return res
}
