package lake

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	DataTag  = "data"
	IndexTag = "index"
	LogTag   = "log"
	StageTag = "staging"
)

var ErrStagingEmpty = errors.New("staging area empty")

type PoolConfig struct {
	Version   int          `zng:"version"`
	Name      string       `zng:"name"`
	ID        ksuid.KSUID  `zng:"id"`
	Layout    order.Layout `zng:"layout"`
	Threshold int64        `zng:"threshold"`
}

type Pool struct {
	PoolConfig
	engine    storage.Engine
	Path      *storage.URI
	DataPath  *storage.URI
	IndexPath *storage.URI
	StagePath *storage.URI
	log       *commit.Log
}

func NewPoolConfig(name string, id ksuid.KSUID, layout order.Layout, thresh int64) *PoolConfig {
	if thresh == 0 {
		thresh = segment.DefaultThreshold
	}
	return &PoolConfig{
		Version:   0,
		Name:      name,
		ID:        id,
		Layout:    layout,
		Threshold: thresh,
	}
}

func (p *PoolConfig) Path(root *storage.URI) *storage.URI {
	return root.AppendPath(p.ID.String())
}

func (p *PoolConfig) Create(ctx context.Context, engine storage.Engine, root *storage.URI) error {
	path := p.Path(root)
	_, err := commit.Create(ctx, engine, LogPath(path), p.Layout.Order)
	return err
}

func (p *PoolConfig) Open(ctx context.Context, engine storage.Engine, root *storage.URI) (*Pool, error) {
	path := p.Path(root)
	log, err := commit.Open(ctx, engine, path.AppendPath(LogTag), p.Layout.Order)
	if err != nil {
		return nil, err
	}
	pool := &Pool{
		PoolConfig: *p,
		engine:     engine,
		Path:       path,
		DataPath:   DataPath(path),
		IndexPath:  IndexPath(path),
		StagePath:  StagePath(path),
		log:        log,
	}
	return pool, nil
}

func (p *PoolConfig) Delete(ctx context.Context, engine storage.Engine, root *storage.URI) error {
	return engine.DeleteByPrefix(ctx, p.Path(root))
}

func (p *Pool) Add(ctx context.Context, r zio.Reader) (ksuid.KSUID, error) {
	w, err := NewWriter(ctx, p)
	if err != nil {
		return ksuid.Nil, err
	}
	err = zio.CopyWithContext(ctx, w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return ksuid.Nil, err
	}
	id := ksuid.New()
	txn := commit.NewAddsTxn(id, w.Segments())
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (p *Pool) Delete(ctx context.Context, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	id := ksuid.New()
	// IDs aren't vetted here and will fail at commit time if problematic.
	txn := commit.NewDeletesTxn(id, ids)
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (p *Pool) Commit(ctx context.Context, id ksuid.KSUID, date nano.Ts, author, message string) error {
	if date == 0 {
		date = nano.Now()
	}
	txn, err := p.LoadFromStaging(ctx, id)
	if err != nil {
		if !zqe.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("commit ID not staged: %s", id)
	}
	txn.AppendCommitMessage(id, date, author, message)
	if err := p.log.Commit(ctx, txn); err != nil {
		return err
	}
	// Commit succeeded.  Delete the staging entry.
	return p.ClearFromStaging(ctx, id)
}

func (p *Pool) Squash(ctx context.Context, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	head, err := p.log.Head(ctx)
	if err != nil {
		if err != journal.ErrEmpty {
			return ksuid.Nil, err
		}
		head = commit.NewSnapshot()
	}
	patch := commit.NewPatch(head)
	for _, id := range ids {
		txn, err := p.LoadFromStaging(ctx, id)
		if err != nil {
			if !zqe.IsNotFound(err) {
				return ksuid.Nil, err
			}
			return ksuid.Nil, fmt.Errorf("commit ID not staged: %s", id)
		}
		commit.Play(patch, txn)
	}
	txn := patch.NewTransaction()
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	for _, id := range ids {
		if err := p.ClearFromStaging(ctx, id); err != nil {
			return ksuid.Nil, err
		}
	}
	return txn.ID, nil
}

func (p *Pool) LookupTags(ctx context.Context, tags []ksuid.KSUID) ([]ksuid.KSUID, error) {
	var ids []ksuid.KSUID
	for _, tag := range tags {
		ok, err := p.SegmentExists(ctx, tag)
		if err != nil {
			return nil, err
		}
		if ok {
			ids = append(ids, tag)
			continue
		}
		snap, ok, err := p.log.SnapshotOfCommit(ctx, 0, tag)
		if err != nil {
			return nil, fmt.Errorf("tag does not exist: %s", tag)
		}
		if !ok {
			return nil, fmt.Errorf("commit tag was previously deleted: %s", tag)
		}
		for _, seg := range snap.SelectAll() {
			ids = append(ids, seg.ID)
		}
	}
	return ids, nil
}

func (p *Pool) SegmentExists(ctx context.Context, id ksuid.KSUID) (bool, error) {
	return p.engine.Exists(ctx, segment.RowObjectPath(p.DataPath, id))
}

func (p *Pool) ClearFromStaging(ctx context.Context, id ksuid.KSUID) error {
	return p.engine.Delete(ctx, p.StagingObject(id))
}

func (p *Pool) ListStagedCommits(ctx context.Context) ([]ksuid.KSUID, error) {
	infos, err := p.engine.List(ctx, p.StagePath)
	if err != nil {
		return nil, err
	}
	ids := make([]ksuid.KSUID, 0, len(infos))
	for _, info := range infos {
		_, name := filepath.Split(info.Name)
		base := strings.TrimSuffix(name, ".zng")
		id, err := ksuid.Parse(base)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (p *Pool) StagingObject(id ksuid.KSUID) *storage.URI {
	return p.StagePath.AppendPath(id.String() + ".zng")
}

func (p *Pool) Log() *commit.Log {
	return p.log
}

func (p *Pool) LoadFromStaging(ctx context.Context, id ksuid.KSUID) (*commit.Transaction, error) {
	return commit.LoadTransaction(ctx, p.engine, id, p.StagingObject(id))
}

func (p *Pool) StoreInStaging(ctx context.Context, txn *commit.Transaction) error {
	b, err := txn.Serialize()
	if err != nil {
		return fmt.Errorf("pool %q: internal error: serialize transaction: %w", p.Name, err)
	}
	return storage.Put(ctx, p.engine, p.StagingObject(txn.ID), bytes.NewReader(b))
}

// ScanStaging writes the staging commits in ids to w.
// If ids is empty, all staging commits are written.
func (p *Pool) ScanStaging(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error {
	if len(ids) == 0 {
		var err error
		ids, err = p.ListStagedCommits(ctx)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return ErrStagingEmpty
		}
	}
	ch := make(chan actions.Interface, 10)
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error)
	go func() {
		var errDone error
		defer func() {
			close(ch)
			if errDone != nil {
				done <- errDone
			}
			close(done)
		}()
		for _, id := range ids {
			txn, err := p.LoadFromStaging(ctx, id)
			if err != nil {
				errDone = err
				return
			}
			for _, action := range txn.Actions {
				select {
				case ch <- action:
				case <-ctx.Done():
					return
				}
			}
			select {
			case ch <- &actions.StagedCommit{Commit: id}:
			case <-ctx.Done():
				return
			}
		}
	}()
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	for p := range ch {
		rec, err := m.MarshalRecord(p)
		if err != nil {
			cancel()
			<-done
			return err
		}
		if err := w.Write(rec); err != nil {
			cancel()
			<-done
			return err
		}
	}
	cancel()
	return <-done
}

func (p *Pool) Scan(ctx context.Context, snap *commit.Snapshot, ch chan segment.Reference) error {
	for _, seg := range snap.SelectAll() {
		select {
		case ch <- *seg:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *Pool) ScanPartitions(ctx context.Context, w zio.Writer, snap *commit.Snapshot, span extent.Span) error {
	ch := make(chan Partition, 10)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var err error
	go func() {
		err = ScanPartitions(ctx, snap, span, p.Layout.Order, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	for p := range ch {
		rec, err := m.MarshalRecord(p)
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return err
}

func (p *Pool) ScanSegments(ctx context.Context, w zio.Writer, snap *commit.Snapshot, span extent.Span) error {
	ch := make(chan segment.Reference)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var err error
	go func() {
		err = ScanSpan(ctx, snap, span, p.Layout.Order, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StylePackage)
	for p := range ch {
		rec, err := m.MarshalRecord(p)
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return err
}

func (p *Pool) IsJournalID(ctx context.Context, id journal.ID) (bool, error) {
	head, tail, err := p.log.Boundaries(ctx)
	if err != nil {
		return false, err
	}
	return id >= tail && id <= head, nil
}

func (p *Pool) ApplyIndexRules(ctx context.Context, rules []index.Rule, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	idxrefs := make([]*index.Reference, 0, len(rules)*len(ids))
	for _, id := range ids {
		//XXX make issue for this.
		// This could be easily parallized with errgroup.
		refs, err := p.indexSegment(ctx, rules, id)
		if err != nil {
			return ksuid.Nil, err
		}
		idxrefs = append(idxrefs, refs...)
	}
	id := ksuid.New()
	txn := commit.NewAddIndicesTxn(id, idxrefs)
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (p *Pool) indexSegment(ctx context.Context, rules []index.Rule, id ksuid.KSUID) ([]*index.Reference, error) {
	r, err := p.engine.Get(ctx, segment.RowObjectPath(p.DataPath, id))
	if err != nil {
		return nil, err
	}
	reader := zngio.NewReader(r, zson.NewContext())
	w, err := index.NewCombiner(ctx, p.engine, p.IndexPath, rules, id)
	if err != nil {
		r.Close()
		return nil, err
	}
	err = zio.CopyWithContext(ctx, w, reader)
	if err != nil {
		w.Abort()
	} else {
		err = w.Close()
	}
	if rerr := r.Close(); err == nil {
		err = rerr
	}
	return w.References(), err
}

type PoolStats struct {
	Size int64 `zng:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `zng:"span"`
}

func (p *Pool) Stats(ctx context.Context, snap *commit.Snapshot) (info PoolStats, err error) {
	ch := make(chan segment.Reference)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err = ScanSpan(ctx, snap, nil, p.Layout.Order, ch)
		close(ch)
	}()
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for segment := range ch {
		info.Size += segment.RowSize
		if poolSpan == nil {
			poolSpan = extent.NewGenericFromOrder(segment.First, segment.Last, p.Layout.Order)
		} else {
			poolSpan.Extend(segment.First)
			poolSpan.Extend(segment.Last)
		}
	}
	//XXX need to change API to take return key range
	if poolSpan != nil {
		min := poolSpan.First()
		if min.Type == zng.TypeTime {
			firstTs, _ := zng.DecodeTime(min.Bytes)
			lastTs, _ := zng.DecodeTime(poolSpan.Last().Bytes)
			if lastTs < firstTs {
				firstTs, lastTs = lastTs, firstTs
			}
			span := nano.NewSpanTs(firstTs, lastTs+1)
			info.Span = &span
		}
	}
	return info, err
}

func DataPath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(DataTag)
}

func StagePath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(StageTag)
}

func LogPath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(LogTag)
}

func IndexPath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(IndexTag)
}
