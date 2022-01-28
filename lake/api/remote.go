package api

import (
	"context"
	"errors"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/api/queryio"
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

func (r *RemoteSession) CreatePool(ctx context.Context, name string, layout order.Layout, seekStride int, thresh int64) (ksuid.KSUID, error) {
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

func (r *RemoteSession) CreateTag(ctx context.Context, poolID ksuid.KSUID, name string, at ksuid.KSUID) error {
	return errors.New("TBD RemoteSession.CreateTag")
}

func (r *RemoteSession) RemoveTag(ctx context.Context, poolID ksuid.KSUID, tagName string) error {
	return errors.New("TBD RemoteSession.RemoveTag")
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

func (r *RemoteSession) Load(ctx context.Context, _ *zed.Context, poolID ksuid.KSUID, branchName string, reader zio.Reader, commit api.CommitMessage) (ksuid.KSUID, error) {
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

type queryReader struct {
	reader zio.Reader
	res    *client.Response
}

var _ zio.Reader = (*queryReader)(nil)

func (q *queryReader) Read() (*zed.Value, error) {
	val, err := q.reader.Read()
	if val == nil || err != nil {
		if _, ok := err.(*zbuf.Control); !ok {
			if closeErr := q.res.Body.Close(); err == nil {
				err = closeErr
			}
		}
	}
	return val, err
}

func (r *RemoteSession) Query(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zio.Reader, error) {
	q, err := r.QueryWithControl(ctx, head, src, srcfiles...)
	if err != nil {
		return nil, err
	}
	return zbuf.NoControl(q), nil
}

func (r *RemoteSession) QueryWithControl(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zbuf.ProgressReader, error) {
	res, err := r.conn.Query(ctx, head, src, srcfiles...)
	if err != nil {
		return nil, err
	}
	reader, err := queryio.NewQuery(res), nil
	if err != nil {
		return nil, err
	}
	return zbuf.MeterReader(&queryReader{reader, res}), nil
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
