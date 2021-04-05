package lake

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	DataTag  = "data"
	LogTag   = "log"
	StageTag = "staging"
)

type PoolConfig struct {
	Version  int            `zng:"version"`
	Name     string         `zng:"name"`
	Id       ksuid.KSUID    `zng:"id"`
	Keys     []field.Static `zng:"keys"`
	Order    zbuf.Order     `zng:"order"`
	Treshold int64          `zng:"threshold"`
}

type Pool struct {
	PoolConfig
	Path      iosrc.URI
	DataPath  iosrc.URI
	StagePath iosrc.URI
	log       *commit.Log
}

func NewPoolConfig(name string, id ksuid.KSUID, keys []field.Static, order zbuf.Order, thresh int64) *PoolConfig {
	if thresh == 0 {
		thresh = segment.DefaultThreshold
	}
	return &PoolConfig{
		Version:  0,
		Name:     name,
		Id:       id,
		Keys:     keys,
		Order:    order,
		Treshold: thresh,
	}
}

func (p *PoolConfig) Path(root iosrc.URI) iosrc.URI {
	return root.AppendPath(p.Id.String())
}

func (p *PoolConfig) Create(ctx context.Context, root iosrc.URI) error {
	path := p.Path(root)
	if err := iosrc.MkdirAll(DataPath(path), 0700); err != nil {
		return err
	}
	if err := iosrc.MkdirAll(StagePath(path), 0700); err != nil {
		return err
	}
	_, err := commit.Create(ctx, LogPath(path), p.Order)
	return err
}

func (p *PoolConfig) Open(ctx context.Context, root iosrc.URI) (*Pool, error) {
	path := p.Path(root)
	log, err := commit.Open(ctx, path.AppendPath("log"), p.Order)
	if err != nil {
		return nil, err
	}
	pool := &Pool{
		PoolConfig: *p,
		Path:       path,
		DataPath:   DataPath(path),
		StagePath:  StagePath(path),
		log:        log,
	}
	return pool, nil
}

func (p *PoolConfig) Delete(ctx context.Context, root iosrc.URI) error {
	return iosrc.RemoveAll(ctx, p.Path(root))
}

func (p *Pool) Add(ctx context.Context, zctx *zson.Context, r zbuf.Reader) (ksuid.KSUID, error) {
	w, err := NewWriter(ctx, p)
	if err != nil {
		return ksuid.Nil, err
	}
	err = zbuf.CopyWithContext(ctx, w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return ksuid.Nil, err
	}
	id := ksuid.New()
	txn := commit.NewAddsTxn(id, w.Segments())
	if err := p.StoreInStaging(ctx, id, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (p *Pool) Commit(ctx context.Context, id ksuid.KSUID, date nano.Ts, author, message string) error {
	if date == 0 {
		date = nano.Now()
	}
	commit, err := p.LoadFromStaging(ctx, id)
	if err != nil {
		if !zqe.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("commit ID not staged: %s", id)
	}
	commit.AppendCommitMessage(id, date, author, message)
	if err := p.log.Commit(ctx, commit); err != nil {
		return err
	}
	// Commit succeeded.  Delete the staging entry.
	return p.ClearFromStaging(ctx, id)
}

func (p *Pool) ClearFromStaging(ctx context.Context, id ksuid.KSUID) error {
	return iosrc.Remove(ctx, p.StagingObject(id))
}

func (p *Pool) GetStagedCommits(ctx context.Context) ([]ksuid.KSUID, error) {
	infos, err := iosrc.ReadDir(ctx, p.StagePath)
	if err != nil {
		return nil, err
	}
	ids := make([]ksuid.KSUID, 0, len(infos))
	for _, info := range infos {
		_, name := filepath.Split(info.Name())
		base := strings.TrimSuffix(name, ".zng")
		id, err := ksuid.Parse(base)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (p *Pool) StagingObject(id ksuid.KSUID) iosrc.URI {
	return p.StagePath.AppendPath(id.String() + ".zng")
}

func (p *Pool) Log() *commit.Log {
	return p.log
}

func (p *Pool) LoadFromStaging(ctx context.Context, id ksuid.KSUID) (commit.Transaction, error) {
	return commit.LoadTransaction(ctx, p.StagingObject(id))
}

func (p *Pool) StoreInStaging(ctx context.Context, id ksuid.KSUID, txn commit.Transaction) error {
	b, err := txn.Serialize()
	if err != nil {
		return fmt.Errorf("pool %q: internal error: serialize transaction: %w", p.Name, err)
	}
	return iosrc.WriteFile(ctx, p.StagingObject(id), b)
}

func DataPath(poolPath iosrc.URI) iosrc.URI {
	return poolPath.AppendPath(DataTag)
}

func StagePath(poolPath iosrc.URI) iosrc.URI {
	return poolPath.AppendPath(StageTag)
}

func LogPath(poolPath iosrc.URI) iosrc.URI {
	return poolPath.AppendPath(LogTag)
}
