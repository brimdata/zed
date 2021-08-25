package api

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type RemoteSession struct {
	conn *client.Connection
}

var _ Interface = (*RemoteSession)(nil)

func OpenRemoteLake(ctx context.Context, host string) (*RemoteSession, error) {
	return &RemoteSession{
		conn: newConnection(host),
	}, nil
}

func NewRemoteWithConnection(conn *client.Connection) *RemoteSession {
	return &RemoteSession{conn}
}

func newConnection(host string) *client.Connection {
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return client.NewConnectionTo(host)
}

func (r *RemoteSession) IDs(ctx context.Context, poolName, branchName string) (ksuid.KSUID, ksuid.KSUID, error) {
	res, err := r.conn.IDs(ctx, poolName, branchName)
	if err != nil {
		return ksuid.Nil, ksuid.Nil, err
	}
	var ids api.IDsResponse
	err = unmarshal(res, &ids)
	return ids.PoolID, ids.BranchID, err
}

func (r *RemoteSession) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (ksuid.KSUID, error) {
	res, err := r.conn.PoolPost(ctx, api.PoolPostRequest{
		Name:   name,
		Layout: layout,
		Thresh: thresh,
	})
	if err != nil {
		return ksuid.Nil, err
	}
	var meta lake.BranchMeta
	err = unmarshal(res, &meta)
	return meta.PoolConfig.ID, err
}

func (r *RemoteSession) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, parent, at ksuid.KSUID) (ksuid.KSUID, error) {
	res, err := r.conn.BranchPost(ctx, poolID, api.BranchPostRequest{
		Name:     name,
		ParentID: parent.String(),
		At:       at.String(),
	})
	if err != nil {
		return ksuid.Nil, err
	}
	var meta lake.BranchMeta
	err = unmarshal(res, &meta)
	return meta.BranchConfig.ID, err
}

func (r *RemoteSession) RemoveBranch(ctx context.Context, poolID, branchID ksuid.KSUID) error {
	return errors.New("TBD RemoteSession.RemoveBranch")
}

func (r *RemoteSession) MergeBranch(ctx context.Context, poolID, branchID, tag ksuid.KSUID) (ksuid.KSUID, error) {
	res, err := r.conn.MergeBranch(ctx, poolID, branchID, tag)
	if err != nil {
		return ksuid.Nil, err
	}
	defer res.Body.Close()
	var body api.CommitResponse
	if err := unmarshal(res, &body); err != nil {
		return ksuid.Nil, err
	}
	return body.Commit, nil
}

func (r *RemoteSession) RemovePool(ctx context.Context, pool ksuid.KSUID) error {
	return r.conn.PoolRemove(ctx, pool)
}

func (r *RemoteSession) RenamePool(ctx context.Context, pool ksuid.KSUID, name string) error {
	if name == "" {
		return errors.New("no pool name provided")
	}
	return r.conn.PoolPut(ctx, pool, api.PoolPutRequest{Name: name})
}

func (r *RemoteSession) Load(ctx context.Context, poolID, branchID ksuid.KSUID, reader zio.Reader, commit api.CommitRequest) (ksuid.KSUID, error) {
	pr, pw := io.Pipe()
	w := zngio.NewWriter(pw, zngio.WriterOpts{})
	go func() {
		zio.CopyWithContext(ctx, w, reader)
		w.Close()
	}()
	rc, err := r.conn.Load(ctx, poolID, branchID, pr, commit)
	if err != nil {
		return ksuid.Nil, err
	}
	var res api.CommitResponse
	if err := unmarshal(rc, &res); err != nil {
		return ksuid.Nil, err
	}
	return res.Commit, err
}

func (r *RemoteSession) AddIndexRules(context.Context, []index.Rule) error {
	return errors.New("unsupported see issue #2934")
}

func (*RemoteSession) DeleteIndexRules(ctx context.Context, ids []ksuid.KSUID) ([]index.Rule, error) {
	return nil, errors.New("unsupported see issue #2934")
}

func (*RemoteSession) ApplyIndexRules(ctx context.Context, rule string, poolID, branchID ksuid.KSUID, inTags []ksuid.KSUID) (ksuid.KSUID, error) {
	return ksuid.Nil, errors.New("unsupported see issue #2934")
}

func (r *RemoteSession) Query(ctx context.Context, d driver.Driver, src string, srcfiles ...string) (zbuf.ScannerStats, error) {
	res, err := r.conn.Query(ctx, src, srcfiles...)
	if err != nil {
		return zbuf.ScannerStats{}, err
	}
	defer res.Body.Close()
	return queryio.RunClientResponse(ctx, d, res)
}

func unmarshal(res *client.Response, i interface{}) error {
	format, err := api.MediaTypeToFormat(res.ContentType)
	if err != nil {
		return err
	}
	//XXX should be ZNG
	zr, err := anyio.NewReaderWithOpts(res.Body, zson.NewContext(), anyio.ReaderOpts{Format: format})
	if err != nil {
		return nil
	}
	rec, err := zr.Read()
	if err != nil {
		return err
	}
	return zson.UnmarshalZNGRecord(rec, i)
}

func (r *RemoteSession) Delete(ctx context.Context, poolID, branchID ksuid.KSUID, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error) {
	res, err := r.conn.Delete(ctx, poolID, branchID, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	defer res.Body.Close()
	var staged api.CommitResponse
	if err := unmarshal(res, &staged); err != nil {
		return ksuid.Nil, err
	}
	return staged.Commit, err
}
