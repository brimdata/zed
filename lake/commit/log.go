package commit

import (
	"context"
	"io"

	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Log struct {
	path      iosrc.URI
	order     zbuf.Order
	journal   *journal.Queue
	snapshots map[journal.ID]*Snapshot
}

const journalHandle = "J"

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

func (l *Log) Commit(ctx context.Context, commit *Transaction) error {
	b, err := commit.Serialize()
	if err != nil {
		return err
	}
	return l.journal.Commit(ctx, b)
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
