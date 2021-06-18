package lakecli

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
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
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const HostEnv = "ZED_LAKE_HOST"

func DefaultHost() string {
	host := os.Getenv(HostEnv)
	if host == "" {
		host = "localhost:9867"
	}
	return host
}

type RemoteFlags struct {
	baseFlags
	host string
}

func NewRemoteFlags(set *flag.FlagSet) Flags {
	l := new(RemoteFlags)
	set.StringVar(&l.host, "host", DefaultHost(), "host[:port] of Zed lake service")
	l.baseFlags.SetFlags(set)
	return l
}

func (l *RemoteFlags) Conn() *client.Connection {
	host := l.host
	if !strings.HasPrefix(l.host, "http") {
		host = "http://" + host
	}
	return client.NewConnectionTo(host)
}

func (l *RemoteFlags) Create(ctx context.Context) (Root, error) {
	return nil, errors.New("cannot create new lake for remove lake")
}

func (l *RemoteFlags) Open(ctx context.Context) (Root, error) {
	return &RemoteRoot{l.Conn()}, nil
}

func (l *RemoteFlags) OpenPool(ctx context.Context) (Pool, error) {
	if l.poolName == "" {
		return nil, errors.New("no pool name provided")
	}
	root, err := l.Open(ctx)
	if err != nil {
		return nil, err
	}
	pool, err := root.LookupPoolByName(ctx, l.poolName)
	if err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, fmt.Errorf("%s: pool not found", l.poolName)
	}
	return root.OpenPool(ctx, pool.ID)
}

func (l *RemoteFlags) CreatePool(ctx context.Context, layout order.Layout, thresh int64) (Pool, error) {
	if l.poolName == "" {
		return nil, errors.New("no pool name provided")
	}
	root, err := l.Open(ctx)
	if err != nil {
		return nil, err
	}
	return root.CreatePool(ctx, l.poolName, layout, thresh)
}

type RemoteRoot struct {
	conn *client.Connection
}

func (r *RemoteRoot) ScanPools(ctx context.Context, zw zio.Writer) error {
	res, err := r.conn.ScanPools(ctx)
	if err != nil {
		return err
	}
	defer res.Close()
	zr := zngio.NewReader(res, zson.NewContext())
	return zio.CopyWithContext(ctx, zw, zr)
}

func (r *RemoteRoot) LookupPoolByName(ctx context.Context, name string) (*lake.PoolConfig, error) {
	res, err := r.conn.ScanPools(ctx)
	if err != nil {
		return nil, nil
	}
	defer res.Close()
	format, err := api.MediaTypeToFormat(res.ContentType)
	if err != nil {
		return nil, err
	}
	zr, err := anyio.NewReaderWithOpts(res, zson.NewContext(), anyio.ReaderOpts{Format: format})
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

func (r *RemoteRoot) OpenPool(ctx context.Context, id ksuid.KSUID) (Pool, error) {
	res, err := r.conn.PoolGet(ctx, id)
	if err != nil {
		return nil, err
	}
	var config lake.PoolConfig
	if err := unmarshal(res, &config); err != nil {
		return nil, err
	}
	return newRemotePool(r.conn, config), nil
}

func (r *RemoteRoot) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (Pool, error) {
	res, err := r.conn.PoolPost(ctx, api.PoolPostRequest{
		Name:   name,
		Layout: layout,
		Thresh: thresh,
	})
	if err != nil {
		return nil, err
	}
	var config lake.PoolConfig
	unmarshal(res, &config)
	return newRemotePool(r.conn, config), nil
}

func (r *RemoteRoot) RemovePool(ctx context.Context, pool ksuid.KSUID) error {
	return r.conn.PoolRemove(ctx, pool)
}

func (r *RemoteRoot) AddIndex(context.Context, []index.Index) error {
	return errors.New("unsupported")
}

func (r *RemoteRoot) DeleteIndices(context.Context, []ksuid.KSUID) ([]index.Index, error) {
	return nil, errors.New("unsupported")
}

func (r *RemoteRoot) LookupIndices(context.Context, []ksuid.KSUID) ([]index.Index, error) {
	return nil, errors.New("unsupported")
}

func (r *RemoteRoot) ScanIndex(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error {
	return errors.New("unsupported")
}

func (r *RemoteRoot) Query(ctx context.Context, d driver.Driver, query string) (zbuf.ScannerStats, error) {
	res, err := r.conn.Query(ctx, query)
	if err != nil {
		return zbuf.ScannerStats{}, err
	}
	defer res.Close()
	return queryio.RunClientResponse(ctx, d, res)
}

type RemotePool struct {
	lake.PoolConfig
	conn *client.Connection
}

func newRemotePool(conn *client.Connection, conf lake.PoolConfig) *RemotePool {
	return &RemotePool{PoolConfig: conf, conn: conn}
}

func unmarshal(r *client.Response, i interface{}) error {
	format, err := api.MediaTypeToFormat(r.ContentType)
	if err != nil {
		return err
	}
	zr, err := anyio.NewReaderWithOpts(r, zson.NewContext(), anyio.ReaderOpts{Format: format})
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	// XXX maybe just have requests that do this request a zson response?
	zw := zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{})
	if err := zio.Copy(zw, zr); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return zson.Unmarshal(buf.String(), i)
}

func (p *RemotePool) Config() lake.PoolConfig {
	return p.PoolConfig
}

func (p *RemotePool) Add(ctx context.Context, r zio.Reader, commit *api.CommitRequest) (ksuid.KSUID, error) {
	pr, pw := io.Pipe()
	w := zngio.NewWriter(pw, zngio.WriterOpts{})
	go func() {
		zio.CopyWithContext(ctx, w, r)
		w.Close()
	}()
	rc, err := p.conn.Add(ctx, p.ID, pr)
	if err != nil {
		return ksuid.Nil, err
	}
	var res api.AddResponse
	if err := unmarshal(rc, &res); err != nil {
		return ksuid.Nil, err
	}
	if commit != nil {
		err = p.conn.Commit(ctx, p.ID, res.Commit, *commit)
	}
	return res.Commit, err
}

func (p *RemotePool) Delete(ctx context.Context, ids []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error) {
	rc, err := p.conn.Delete(ctx, p.ID, ids)
	if err != nil {
		return ksuid.Nil, err
	}
	defer rc.Close()
	var res api.StagedCommit
	if err := unmarshal(rc, &res); err != nil {
		return ksuid.Nil, err
	}
	if commit != nil {
		err = p.Commit(ctx, res.Commit, *commit)
	}
	return res.Commit, err
}

func (p *RemotePool) Commit(ctx context.Context, id ksuid.KSUID, commit api.CommitRequest) error {
	return p.conn.Commit(ctx, p.ID, id, commit)
}

func (p *RemotePool) Squash(ctx context.Context, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	rc, err := p.conn.Squash(ctx, p.ID, ids)
	if err != nil {
		return ksuid.Nil, err
	}
	defer rc.Close()
	var res api.StagedCommit
	if err := unmarshal(rc, &res); err != nil {
		return ksuid.Nil, err
	}
	return res.Commit, nil
}

func (p *RemotePool) ScanStaging(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error {
	res, err := p.conn.ScanStaging(ctx, p.ID, ids)
	if err != nil {
		return err
	}
	defer res.Close()
	if res.StatusCode == http.StatusNoContent {
		return lake.ErrStagingEmpty
	}
	zr := zngio.NewReader(res, zson.NewContext())
	return zio.CopyWithContext(ctx, w, zr)

}

func (p *RemotePool) ScanLog(ctx context.Context, w zio.Writer, head, tail journal.ID) error {
	res, err := p.conn.ScanLog(ctx, p.ID)
	if err != nil {
		return err
	}
	defer res.Close()
	zr := zngio.NewReader(res, zson.NewContext())
	return zio.CopyWithContext(ctx, w, zr)
}

func (p *RemotePool) ScanSegments(ctx context.Context, w zio.Writer, at string, partitions bool, span extent.Span) error {
	// TODO add segments to connection.PoolScanSegments
	res, err := p.conn.ScanSegments(ctx, p.ID, at, partitions)
	if err != nil {
		return err
	}
	defer res.Close()
	zr := zngio.NewReader(res, zson.NewContext())
	return zio.CopyWithContext(ctx, w, zr)
}

func (p *RemotePool) Index(ctx context.Context, rules []index.Index, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	return ksuid.Nil, errors.New("unsupported")
}
