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
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Log struct {
	path      iosrc.URI
	order     zbuf.Order
	journal   *journal.Queue
	snapshots map[journal.ID]*Snapshot
}

const (
	journalHandle = "J"
	maxRetries    = 10
)

var ErrRetriesExceeded = fmt.Errorf("commit journal unavailable after %d attempts", maxRetries)

func newLog(path iosrc.URI, order zbuf.Order) *Log {
	return &Log{
		path:      path,
		order:     order,
		snapshots: make(map[journal.ID]*Snapshot),
	}
}

func Open(ctx context.Context, path iosrc.URI, order zbuf.Order) (*Log, error) {
	l := newLog(path, order)
	var err error
	l.journal, err = journal.Open(ctx, l.path.AppendPath(journalHandle))
	if err != nil {
		return nil, err
	}
	return l, nil
}

func Create(ctx context.Context, path iosrc.URI, order zbuf.Order) (*Log, error) {
	l := newLog(path, order)
	j, err := journal.Create(ctx, l.path.AppendPath(journalHandle))
	if err != nil {
		return nil, err
	}
	l.journal = j
	return l, nil
}

func (l *Log) Boundaries(ctx context.Context) (journal.ID, journal.ID, error) {
	return l.journal.Boundaries(ctx)
}

func (l *Log) Commit(ctx context.Context, commit *Transaction) error {
	b, err := commit.Serialize()
	if err != nil {
		return err
	}
	//XXX It's a bug to do this loop here as the committer above should
	// recompute its commit and check for a write-conflict.  Right now,
	// we are just demo-ing concurrent loads so it's not a problem,
	// but it will eventually become one.  This is all addressed in #2546.
	for attempts := 0; attempts < maxRetries; attempts++ {
		err := l.journal.Commit(ctx, b)
		if err != nil {
			if os.IsExist(err) {
				time.Sleep(time.Millisecond)
				continue
			}
			return err
		}
		return nil
	}
	return ErrRetriesExceeded
}

func (l *Log) Open(ctx context.Context, head, tail journal.ID) (io.Reader, error) {
	if head == journal.Nil {
		var err error
		head, err = l.journal.ReadHead(ctx)
		if err != nil {
			return nil, err
		}
		if head == journal.Nil {
			return nil, journal.ErrEmpty
		}
	}
	if tail == journal.Nil {
		var err error
		tail, err = l.journal.ReadTail(ctx)
		if err != nil {
			return nil, err
		}
	}
	return l.journal.NewReader(ctx, head, tail), nil
}

func (l *Log) OpenAsZNG(ctx context.Context, head, tail journal.ID) (*zngio.Reader, error) {
	r, err := l.Open(ctx, head, tail)
	if err != nil {
		return nil, err
	}
	return zngio.NewReader(r, zson.NewContext()), nil
}

func (l *Log) Head(ctx context.Context) (*Snapshot, error) {
	return l.Snapshot(ctx, 0)
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
	snapshot := newSnapshotAt(at)
	reader := actions.NewDeserializer(r)
	for {
		action, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if action == nil {
			break
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
	var ok bool
	snapshot := newSnapshotAt(at)
	reader := actions.NewDeserializer(r)
	for {
		action, err := reader.Read()
		if err != nil {
			return nil, false, err
		}
		if action == nil {
			return snapshot, ok, nil
		}
		if action.CommitID() == commit {
			ok = true
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
	tail, err := l.journal.ReadTail(ctx)
	if err != nil {
		return journal.Nil, err
	}
	for cursor := at; cursor >= tail; cursor-- {
		b, err := l.journal.Load(ctx, cursor)
		if err != nil {
			return journal.Nil, err
		}
		reader := actions.NewDeserializer(bytes.NewReader(b))
		action, err := reader.Read()
		if err != nil {
			return journal.Nil, err
		}
		if action == nil {
			break
		}
		if action.CommitID() == commit {
			return cursor, nil
		}
	}
	return journal.Nil, ErrNotFound
}
