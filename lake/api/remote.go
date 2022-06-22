package api

import (
	"context"
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/segmentio/ksuid"
)

type remote struct {
	conn *client.Connection
}

var _ Interface = (*remote)(nil)

func NewRemoteLake(conn *client.Connection) Interface {
	return &remote{conn}
}

func (l *remote) Root() *lake.Root {
	return nil
}

func (r *remote) PoolID(ctx context.Context, poolName string) (ksuid.KSUID, error) {
	config, err := LookupPoolByName(ctx, r, poolName)
	if err != nil {
		return ksuid.Nil, err
	}
	return config.ID, nil
}

func (r *remote) CommitObject(ctx context.Context, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	res, err := r.conn.BranchGet(ctx, poolID, branchName)
	return res.Commit, err
}

func (r *remote) CreatePool(ctx context.Context, name string, layout order.Layout, seekStride int, thresh int64) (ksuid.KSUID, error) {
	res, err := r.conn.CreatePool(ctx, api.PoolPostRequest{
		Name:       name,
		Layout:     layout,
		SeekStride: seekStride,
		Thresh:     thresh,
	})
	if err != nil {
		return ksuid.Nil, err
	}
	return res.Pool.ID, err
}

func (r *remote) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, at ksuid.KSUID) error {
	_, err := r.conn.CreateBranch(ctx, poolID, api.BranchPostRequest{
		Name:   name,
		Commit: at.String(),
	})
	return err
}

func (r *remote) RemoveBranch(ctx context.Context, poolID ksuid.KSUID, branchName string) error {
	return errors.New("TBD remote.RemoveBranch")
}

func (r *remote) MergeBranch(ctx context.Context, poolID ksuid.KSUID, childBranch, parentBranch string, message api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.MergeBranch(ctx, poolID, childBranch, parentBranch, message)
	return res.Commit, err
}

func (r *remote) Compact(ctx context.Context, poolID ksuid.KSUID, branch string, objects []ksuid.KSUID, commit api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.Compact(ctx, poolID, branch, objects, commit)
	return res.Commit, err
}

func (r *remote) RemovePool(ctx context.Context, pool ksuid.KSUID) error {
	return r.conn.RemovePool(ctx, pool)
}

func (r *remote) RenamePool(ctx context.Context, pool ksuid.KSUID, name string) error {
	if name == "" {
		return errors.New("no pool name provided")
	}
	return r.conn.RenamePool(ctx, pool, api.PoolPutRequest{Name: name})
}

func (r *remote) Load(ctx context.Context, _ *zed.Context, poolID ksuid.KSUID, branchName string, reader zio.Reader, commit api.CommitMessage) (ksuid.KSUID, error) {
	pr, pw := io.Pipe()
	go func() {
		w := zngio.NewWriter(pw)
		zio.CopyWithContext(ctx, w, reader)
		w.Close()
	}()
	res, err := r.conn.Load(ctx, poolID, branchName, pr, commit)
	return res.Commit, err
}

func (r *remote) Revert(ctx context.Context, poolID ksuid.KSUID, branchName string, commitID ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.Revert(ctx, poolID, branchName, commitID, message)
	return res.Commit, err
}

func (r *remote) Query(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zio.ReadCloser, error) {
	q, err := r.QueryWithControl(ctx, head, src, srcfiles...)
	if err != nil {
		return nil, err
	}
	return zio.NewReadCloser(zbuf.NoControl(q), q), nil
}

func (r *remote) QueryWithControl(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zbuf.ProgressReadCloser, error) {
	res, err := r.conn.Query(ctx, head, src, srcfiles...)
	if err != nil {
		return nil, err
	}
	q, err := queryio.NewQuery(res.Body), nil
	if err != nil {
		return nil, err
	}
	return zbuf.MeterReadCloser(q), nil
}

func (r *remote) Delete(ctx context.Context, poolID ksuid.KSUID, branchName string, tags []ksuid.KSUID, commit api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.Delete(ctx, poolID, branchName, tags, commit)
	return res.Commit, err
}

func (r *remote) DeleteByPredicate(ctx context.Context, poolID ksuid.KSUID, branchName, src string, commit api.CommitMessage) (ksuid.KSUID, error) {
	res, err := r.conn.DeleteByPredicate(ctx, poolID, branchName, src, commit)
	return res.Commit, err
}

func (r *remote) AddIndexRules(ctx context.Context, rules []index.Rule) error {
	return r.conn.AddIndexRules(ctx, rules)
}

func (r *remote) DeleteIndexRules(ctx context.Context, ids []ksuid.KSUID) ([]index.Rule, error) {
	res, err := r.conn.DeleteIndexRules(ctx, ids)
	return res.Rules, err
}

func (r *remote) ApplyIndexRules(ctx context.Context, rule string, poolID ksuid.KSUID, branchName string, inTags []ksuid.KSUID) (ksuid.KSUID, error) {
	res, err := r.conn.ApplyIndexRules(ctx, poolID, branchName, rule, inTags)
	return res.Commit, err
}

func (r *remote) UpdateIndex(ctx context.Context, rules []string, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	res, err := r.conn.UpdateIndex(ctx, poolID, branchName, rules)
	return res.Commit, err
}

func (r *remote) AddVectors(ctx context.Context, pool ksuid.KSUID, branch string, objects []ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	panic("TBD")
}

func (r *remote) DeleteVectors(ctx context.Context, poolID ksuid.KSUID, branchName string, ids []ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	panic("TBD")
}
