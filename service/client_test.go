package service_test

import (
	"bytes"
	"io"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/superdb/super"
	"github.com/superdb/super/api"
	"github.com/superdb/super/api/client"
	"github.com/superdb/super/compiler/srcfiles"
	"github.com/superdb/super/db"
	dbapi "github.com/superdb/super/db/api"
	"github.com/superdb/super/db/branches"
	"github.com/superdb/super/db/tables"
	"github.com/superdb/super/runtime/exec"
	"github.com/superdb/super/sbuf"
	"github.com/superdb/super/sio"
	"github.com/superdb/super/sio/bsupio"
	"github.com/superdb/super/sio/supio"
)

type testClient struct {
	*testing.T
	*client.Connection
}

func (c *testClient) TestTableStats(id uuid.UUID) exec.TableStats {
	r, err := c.Connection.TableStats(c.Context(), id)
	require.NoError(c, err)
	return r
}

func (c *testClient) TestTableGet(id uuid.UUID) (config tables.Config) {
	remote := dbapi.NewRemoteDB(c.Connection)
	table, err := dbapi.LookupTableByID(c.Context(), remote, id)
	require.NoError(c, err)
	return *table
}

func (c *testClient) TestBranchGet(id uuid.UUID) (config db.BranchMeta) {
	remote := dbapi.NewRemoteDB(c.Connection)
	branch, err := dbapi.LookupBranchByID(c.Context(), remote, id)
	require.NoError(c, err)
	return *branch
}

func (c *testClient) TestTableList() []tables.Config {
	r, err := c.Query(c.Context(), srcfiles.Plain("from :tables"))
	require.NoError(c, err)
	defer r.Body.Close()
	var configs []tables.Config
	puller, err := bsupio.NewReader(c.Context(), super.NewContext(), r.Body, nil, nil, 1)
	require.NoError(c, err)
	reader := sbuf.PullerReader(sbuf.NewMaterializer(puller))
	for {
		rec, err := reader.Read()
		require.NoError(c, err)
		if rec == nil {
			return configs
		}
		var config tables.Config
		err = super.Unmarshal(*rec, &config)
		require.NoError(c, err)
		configs = append(configs, config)
	}
}

func (c *testClient) TestTablePost(payload api.TablePostRequest) uuid.UUID {
	r, err := c.Connection.CreateTable(c.Context(), payload)
	require.NoError(c, err)
	return r.Table.ID
}

func (c *testClient) TestBranchPost(tableID uuid.UUID, payload api.BranchPostRequest) branches.Config {
	r, err := c.Connection.CreateBranch(c.Context(), tableID, payload)
	require.NoError(c, err)
	return r
}

func (c *testClient) TestQuery(query string) string {
	r, err := c.Connection.Query(c.Context(), srcfiles.Plain(query))
	require.NoError(c, err)
	defer r.Body.Close()
	zr, err := bsupio.NewReader(c.Context(), super.NewContext(), r.Body, nil, nil, 1)
	require.NoError(c, err)
	var buf bytes.Buffer
	zw := supio.NewWriter(sio.NopCloser(&buf), supio.WriterOpts{})
	require.NoError(c, sio.Copy(zw, sbuf.PullerReader(sbuf.NewMaterializer(zr))))
	return buf.String()
}

func (c *testClient) TestLoad(tableID uuid.UUID, branchName string, r io.Reader) uuid.UUID {
	commit, err := c.Connection.Load(c.Context(), tableID, branchName, "", r, api.CommitMessage{})
	require.NoError(c, err)
	return commit.Commit
}

func (c *testClient) TestAuthMethod() api.AuthMethodResponse {
	r, err := c.Connection.AuthMethod(c.Context())
	require.NoError(c, err)
	return r
}

func (c *testClient) TestAuthIdentity() api.AuthIdentityResponse {
	r, err := c.Connection.AuthIdentity(c.Context())
	require.NoError(c, err)
	return r
}
