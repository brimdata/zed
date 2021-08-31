package commit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Log struct {
	path      *storage.URI
	order     order.Which
	journal   *journal.Queue
	snapshots map[journal.ID]*Snapshot
}

const (
	maxRetries = 10
)

var ErrRetriesExceeded = fmt.Errorf("commit journal unavailable after %d attempts", maxRetries)

func newLog(path *storage.URI, o order.Which) *Log {
	return &Log{
		path:      path,
		order:     o,
		snapshots: make(map[journal.ID]*Snapshot),
	}
}

func Open(ctx context.Context, engine storage.Engine, path *storage.URI, o order.Which) (*Log, error) {
	l := newLog(path, o)
	var err error
	l.journal, err = journal.Open(ctx, engine, l.path)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func Create(ctx context.Context, engine storage.Engine, path *storage.URI, o order.Which, base journal.ID) (*Log, error) {
	l := newLog(path, o)
	j, err := journal.Create(ctx, engine, l.path, base)
	if err != nil {
		return nil, err
	}
	l.journal = j
	return l, nil
}

func (l *Log) Boundaries(ctx context.Context) (journal.ID, journal.ID, error) {
	return l.journal.Boundaries(ctx)
}

func (l *Log) Commit(ctx context.Context, commit *Transaction) (journal.ID, error) {
	b, err := commit.Serialize()
	if err != nil {
		return journal.Nil, err
	}
	//XXX It's a bug to do this loop here as the committer above should
	// recompute its commit and check for a write-conflict.  Right now,
	// we are just demo-ing concurrent loads so it's not a problem,
	// but it will eventually become one.  This is all addressed in #2546.
	for attempts := 0; attempts < maxRetries; attempts++ {
		tip, err := l.journal.Commit(ctx, b)
		if err != nil {
			if os.IsExist(err) {
				time.Sleep(time.Millisecond)
				continue
			}
			return journal.Nil, err
		}
		return tip, nil
	}
	return journal.Nil, ErrRetriesExceeded
}

func (l *Log) Open(ctx context.Context, head, tail journal.ID) (io.Reader, error) {
	return l.journal.Open(ctx, head, tail)
}

func (l *Log) OpenAsZNG(ctx context.Context, zctx *zson.Context, head, tail journal.ID) (*zngio.Reader, error) {
	return l.journal.OpenAsZNG(ctx, zctx, head, tail)
}

func (l *Log) Tip(ctx context.Context) (*Snapshot, error) {
	return l.Snapshot(ctx, 0)
}

func (l *Log) MoveTail(ctx context.Context, tail, base journal.ID) error {
	return l.journal.MoveTail(ctx, tail, base)
}

func (l *Log) ReadTail(ctx context.Context) (journal.ID, journal.ID, error) {
	return l.journal.ReadTail(ctx)
}

func badEntry(entry interface{}) error {
	return fmt.Errorf("internal error: corrupt journal has unknown entry type %T", entry)
}

func (l *Log) Snapshot(ctx context.Context, at journal.ID) (*Snapshot, error) {
	if at == journal.Nil {
		var err error
		at, err = l.journal.ReadHead(ctx)
		if err != nil {
			return nil, err
		}
	}
	if snap, ok := l.snapshots[at]; ok {
		return snap, nil
	}
	r, err := l.Open(ctx, at, journal.Nil)
	if err != nil {
		return nil, err
	}
	snapshot := NewSnapshot()
	reader := zngbytes.NewDeserializer(r, actions.JournalTypes)
	for {
		entry, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if entry == nil {
			break
		}
		action, ok := entry.(actions.Interface)
		if !ok {
			return nil, badEntry(entry)
		}
		PlayAction(snapshot, action)
	}
	l.snapshots[at] = snapshot
	return snapshot, nil
}

func (l *Log) SnapshotOfCommit(ctx context.Context, at journal.ID, commit ksuid.KSUID) (*Snapshot, bool, error) {
	if at == journal.Nil {
		var err error
		at, err = l.journal.ReadHead(ctx)
		if err != nil {
			return nil, false, err
		}
	}
	r, err := l.Open(ctx, at, journal.Nil)
	if err != nil {
		return nil, false, err
	}
	var valid bool
	snapshot := NewSnapshot()
	reader := zngbytes.NewDeserializer(r, actions.JournalTypes)
	for {
		entry, err := reader.Read()
		if err != nil {
			return nil, false, err
		}
		if entry == nil {
			return snapshot, valid, nil
		}
		action, ok := entry.(actions.Interface)
		if !ok {
			return nil, false, badEntry(entry)
		}
		if action.CommitID() == commit {
			valid = true
			PlayAction(snapshot, action)
			continue
		}
		if del, ok := action.(*actions.Delete); ok && snapshot.Exists(del.ID) {
			PlayAction(snapshot, action)
		}
	}
}

func (l *Log) JournalIDOfCommit(ctx context.Context, at journal.ID, commit ksuid.KSUID) (journal.ID, error) {
	if at == journal.Nil {
		var err error
		at, err = l.journal.ReadHead(ctx)
		if err != nil {
			return journal.Nil, err
		}
	}
	tail, _, err := l.journal.ReadTail(ctx)
	if err != nil {
		return journal.Nil, err
	}
	for cursor := at; cursor >= tail; cursor-- {
		b, err := l.journal.Load(ctx, cursor)
		if err != nil {
			return journal.Nil, err
		}
		reader := zngbytes.NewDeserializer(bytes.NewReader(b), actions.JournalTypes)
		entry, err := reader.Read()
		if err != nil {
			return journal.Nil, err
		}
		if entry == nil {
			break
		}
		action, ok := entry.(actions.Interface)
		if !ok {
			return journal.Nil, badEntry(entry)
		}
		if action.CommitID() == commit {
			return cursor, nil
		}
	}
	return journal.Nil, ErrNotFound
}

func (l *Log) TipOfJournal(ctx context.Context) (journal.ID, error) {
	return l.journal.ReadHead(ctx)
}
