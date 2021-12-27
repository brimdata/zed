package service_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/lake"
	lakeapi "github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	*testing.T
	*client.Connection
}

func (c *testClient) TestPoolStats(id ksuid.KSUID) lake.PoolStats {
	r, err := c.Connection.PoolStats(context.Background(), id)
	require.NoError(c, err)
	return r
}

func (c *testClient) TestPoolGet(id ksuid.KSUID) (config pools.Config) {
	remote := lakeapi.NewRemoteWithConnection(c.Connection)
	pool, err := lakeapi.LookupPoolByID(context.Background(), remote, id)
	require.NoError(c, err)
	return *pool
}

func (c *testClient) TestBranchGet(id ksuid.KSUID) (config lake.BranchMeta) {
	remote := lakeapi.NewRemoteWithConnection(c.Connection)
	branch, err := lakeapi.LookupBranchByID(context.Background(), remote, id)
	require.NoError(c, err)
	return *branch
}

func (c *testClient) TestPoolList() []pools.Config {
	r, err := c.Query(context.Background(), nil, "from :pools")
	require.NoError(c, err)
	defer r.Body.Close()
	var confs []pools.Config
	zr := zngio.NewReader(r.Body, zed.NewContext())
	for {
		rec, err := zr.Read()
		require.NoError(c, err)
		if rec == nil {
			return confs
		}
		var pool pools.Config
		err = zson.UnmarshalZNGRecord(rec, &pool)
		require.NoError(c, err)
		confs = append(confs, pool)
	}
}

func (c *testClient) TestPoolPost(payload api.PoolPostRequest) ksuid.KSUID {
	r, err := c.Connection.CreatePool(context.Background(), payload)
	require.NoError(c, err)
	return r.Pool.ID
}

func (c *testClient) TestBranchPost(poolID ksuid.KSUID, payload api.BranchPostRequest) branches.Config {
	r, err := c.Connection.CreateBranch(context.Background(), poolID, payload)
	require.NoError(c, err)
	return r
}

func (c *testClient) TestQuery(query string) string {
	r, err := c.Connection.Query(context.Background(), nil, query)
	require.NoError(c, err)
	defer r.Body.Close()
	zr := zngio.NewReader(r.Body, zed.NewContext())
	var buf bytes.Buffer
	zw := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	require.NoError(c, zio.Copy(zw, zr))
	return buf.String()
}

func (c *testClient) TestLoad(poolID ksuid.KSUID, branchName string, r io.Reader) {
	_, err := c.Connection.Load(context.Background(), poolID, branchName, r, api.CommitMessage{})
	require.NoError(c, err)
}

func (c *testClient) TestAuthMethod() api.AuthMethodResponse {
	r, err := c.Connection.AuthMethod(context.Background())
	require.NoError(c, err)
	return r
}

func (c *testClient) TestAuthIdentity() api.AuthIdentityResponse {
	r, err := c.Connection.AuthIdentity(context.Background())
	require.NoError(c, err)
	return r
}
