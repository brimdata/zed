package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/client"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var _ Interface = (*RemoteSession)(nil)

type RemoteSession struct {
	conn *client.Connection
}

func OpenRemoteLake(ctx context.Context, host string) (*RemoteSession, error) {
	return &RemoteSession{
		conn: newConnection(host),
	}, nil
}

func newConnection(host string) *client.Connection {
	if !strings.HasPrefix(host, "http") {
		host = "http://" + host
	}
	return client.NewConnectionTo(host)
}

func (r *RemoteSession) ScanPools(ctx context.Context, zw zio.Writer) error {
	res, err := r.conn.ScanPools(ctx)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	zr := zngio.NewReader(res.Body, zson.NewContext())
	return zio.CopyWithContext(ctx, zw, zr)
}

func (r *RemoteSession) LookupPool(ctx context.Context, name string) (*lake.PoolConfig, error) {
	res, err := r.conn.ScanPools(ctx)
	if err != nil {
		return nil, nil
	}
	defer res.Body.Close()
	format, err := api.MediaTypeToFormat(res.ContentType)
	if err != nil {
		return nil, err
	}
	zr, err := anyio.NewReaderWithOpts(res.Body, zson.NewContext(), anyio.ReaderOpts{Format: format})
	if err != nil {
		return nil, nil
	}
	for {
		rec, err := zr.Read()
		if rec == nil || err != nil {
			return nil, err
		}
		var pool lake.PoolConfig
		if err := zson.UnmarshalZNGRecord(rec, &pool); err != nil {
			return nil, err
		}
		if pool.Name == name {
			return &pool, nil
		}
	}
}

func (r *RemoteSession) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (*lake.PoolConfig, error) {
	res, err := r.conn.PoolPost(ctx, api.PoolPostRequest{
		Name:   name,
		Layout: layout,
		Thresh: thresh,
	})
	if err != nil {
		return nil, err
	}
	var config lake.PoolConfig
	err = unmarshal(res, &config)
	return &config, err
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

func (r *RemoteSession) Add(ctx context.Context, poolID ksuid.KSUID, reader zio.Reader, commit *api.CommitRequest) (ksuid.KSUID, error) {
	pr, pw := io.Pipe()
	w := zngio.NewWriter(pw, zngio.WriterOpts{})
	go func() {
		zio.CopyWithContext(ctx, w, reader)
		w.Close()
	}()
	rc, err := r.conn.Add(ctx, poolID, pr)
	if err != nil {
		return ksuid.Nil, err
	}
	var res api.AddResponse
	if err := unmarshal(rc, &res); err != nil {
		return ksuid.Nil, err
	}
	if commit != nil {
		err = r.conn.Commit(ctx, poolID, res.Commit, *commit)
	}
	return res.Commit, err
}

func (r *RemoteSession) AddIndexRules(context.Context, []index.Rule) error {
	return errors.New("unsupported see issue #2934")
}

func (*RemoteSession) DeleteIndexRules(ctx context.Context, ids []ksuid.KSUID) ([]index.Rule, error) {
	return nil, errors.New("unsupported see issue #2934")
}

func (*RemoteSession) ApplyIndexRules(ctx context.Context, rule string, poolID ksuid.KSUID, inTags []ksuid.KSUID) (ksuid.KSUID, error) {
	return ksuid.Nil, errors.New("unsupported see issue #2934")
}

func (r *RemoteSession) ScanIndexRules(ctx context.Context, w zio.Writer, names []string) error {
	return errors.New("unsupported see issue #2934")
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

func (r *RemoteSession) Commit(ctx context.Context, poolID, id ksuid.KSUID, commit api.CommitRequest) error {
	return r.conn.Commit(ctx, poolID, id, commit)
}

func (r *RemoteSession) Delete(ctx context.Context, poolID ksuid.KSUID, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error) {
	res, err := r.conn.Delete(ctx, poolID, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	defer res.Body.Close()
	var staged api.StagedCommit
	if err := unmarshal(res, &staged); err != nil {
		return ksuid.Nil, err
	}
	if commit != nil {
		err = r.Commit(ctx, poolID, staged.Commit, *commit)
	}
	return staged.Commit, err
}

func (r *RemoteSession) DeleteFromStaging(ctx context.Context, poolID ksuid.KSUID, id ksuid.KSUID) error {
	return r.conn.DeleteFromStaging(ctx, poolID, id)
}

func (r *RemoteSession) Squash(ctx context.Context, poolID ksuid.KSUID, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	res, err := r.conn.Squash(ctx, poolID, ids)
	if err != nil {
		return ksuid.Nil, err
	}
	defer res.Body.Close()
	var staged api.StagedCommit
	if err := unmarshal(res, &staged); err != nil {
		return ksuid.Nil, err
	}
	return staged.Commit, nil
}

func (r *RemoteSession) ScanStaging(ctx context.Context, poolID ksuid.KSUID, w zio.Writer, ids []ksuid.KSUID) error {
	res, err := r.conn.ScanStaging(ctx, poolID, ids)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		return lake.ErrStagingEmpty
	}
	zr := zngio.NewReader(res.Body, zson.NewContext())
	return zio.CopyWithContext(ctx, w, zr)

}

func (r *RemoteSession) ScanLog(ctx context.Context, poolID ksuid.KSUID, w zio.Writer, head, tail journal.ID) error {
	res, err := r.conn.ScanLog(ctx, poolID)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	zr := zngio.NewReader(res.Body, zson.NewContext())
	return zio.CopyWithContext(ctx, w, zr)
}

func (r *RemoteSession) ScanSegments(ctx context.Context, poolID ksuid.KSUID, w zio.Writer, at ksuid.KSUID, partitions bool, span extent.Span) (err error) {
	res, err := r.conn.ScanSegments(ctx, poolID, at, partitions)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	zr := zngio.NewReader(res.Body, zson.NewContext())
	return zio.CopyWithContext(ctx, w, zr)
}
