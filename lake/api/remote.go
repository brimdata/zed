package api

import (
	"context"
	"errors"
	"io"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/segmentio/ksuid"
)

type RemoteSession struct {
	conn *client.Connection
}

var _ Interface = (*RemoteSession)(nil)

func OpenRemoteLake(ctx context.Context, url string) (*RemoteSession, error) {
	return &RemoteSession{
		conn: client.NewConnectionTo(url),
	}, nil
}

func NewRemoteWithConnection(conn *client.Connection) *RemoteSession {
	return &RemoteSession{conn}
}

func (r *RemoteSession) PoolID(ctx context.Context, poolName string) (ksuid.KSUID, error) {
	config, err := LookupPoolByName(ctx, r, poolName)
	if err != nil {
		return ksuid.Nil, err
	}
	return config.ID, nil
}

func (r *RemoteSession) CommitObject(ctx context.Context, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	res, err := r.conn.BranchGet(ctx, poolID, branchName)
	return res.Commit, err
}

func (r *RemoteSession) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (ksuid.KSUID, error) {
	res, err := r.conn.CreatePool(ctx, api.PoolPostRequest{
		Name:   name,
		Layout: layout,
		Thresh: thresh,
	})
	if err != nil {
		return ksuid.Nil, err
	}
	return res.Pool.ID, err
}

func (r *RemoteSession) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, at ksuid.KSUID) error {
	_, err := r.conn.CreateBranch(ctx, poolID, api.BranchPostRequest{
		Name:   name,
		Commit: at.String(),
	})
	return err
}

func (r *RemoteSession) RemoveBranch(ctx context.Context, poolID ksuid.KSUID, branchName string) error {
	return errors.New("TBD RemoteSession.RemoveBranch")
}

func (r *RemoteSession) MergeBranch(ctx context.Context, poolID ksuid.KSUID, childBranch, parentBranch string, message api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.MergeBranch(ctx, poolID, childBranch, parentBranch, message)
	return res.Commit, err
}

func (r *RemoteSession) RemovePool(ctx context.Context, pool ksuid.KSUID) error {
	return r.conn.RemovePool(ctx, pool)
}

func (r *RemoteSession) RenamePool(ctx context.Context, pool ksuid.KSUID, name string) error {
	if name == "" {
		return errors.New("no pool name provided")
	}
	return r.conn.RenamePool(ctx, pool, api.PoolPutRequest{Name: name})
}

func (r *RemoteSession) Load(ctx context.Context, poolID ksuid.KSUID, branchName string, reader zio.Reader, commit api.CommitMessage) (ksuid.KSUID, error) {
	pr, pw := io.Pipe()
	w := zngio.NewWriter(pw, zngio.WriterOpts{LZ4BlockSize: zngio.DefaultLZ4BlockSize})
	go func() {
		zio.CopyWithContext(ctx, w, reader)
		w.Close()
	}()
	res, err := r.conn.Load(ctx, poolID, branchName, pr, commit)
	return res.Commit, err
}

func (r *RemoteSession) Revert(ctx context.Context, poolID ksuid.KSUID, branchName string, commitID ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.Revert(ctx, poolID, branchName, commitID, message)
	return res.Commit, err
}

func (r *RemoteSession) Query(ctx context.Context, d driver.Driver, head *lakeparse.Commitish, src string, srcfiles ...string) (zbuf.ScannerStats, error) {
	res, err := r.conn.Query(ctx, head, src, srcfiles...)
	if err != nil {
		return zbuf.ScannerStats{}, err
	}
	defer res.Body.Close()
	return queryio.RunClientResponse(ctx, d, res)
}

func (r *RemoteSession) Delete(ctx context.Context, poolID ksuid.KSUID, branchName string, tags []ksuid.KSUID, commit api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.Delete(ctx, poolID, branchName, tags, commit)
	return res.Commit, err
}

func (r *RemoteSession) AddIndexRules(ctx context.Context, rules []index.Rule) error {
	return r.conn.AddIndexRules(ctx, rules)
}

func (r *RemoteSession) DeleteIndexRules(ctx context.Context, ids []ksuid.KSUID) ([]index.Rule, error) {
	res, err := r.conn.DeleteIndexRules(ctx, ids)
	return res.Rules, err
}

func (r *RemoteSession) ApplyIndexRules(ctx context.Context, rule string, poolID ksuid.KSUID, branchName string, inTags []ksuid.KSUID) (ksuid.KSUID, error) {
	res, err := r.conn.ApplyIndexRules(ctx, poolID, branchName, rule, inTags)
	return res.Commit, err
}

func (r *RemoteSession) UpdateIndex(ctx context.Context, rules []string, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	res, err := r.conn.UpdateIndex(ctx, poolID, branchName, rules)
	return res.Commit, err
}
