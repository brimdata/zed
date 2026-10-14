package op

import (
	"uuid"

	"github.com/superdb/super"
	"github.com/superdb/super/db"
	"github.com/superdb/super/runtime"
	"github.com/superdb/super/sbuf"
	"github.com/superdb/super/vector"
	"github.com/superdb/super/vector/vio"
)

type Load struct {
	rctx    *runtime.Context
	root    *db.Root
	parent  vio.Puller
	tableID uuid.UUID
	branch  string
	author  string
	message string
	meta    string
	done    bool
}

func NewLoad(rctx *runtime.Context, root *db.Root, parent vio.Puller, tableID uuid.UUID, branch, author, message, meta string) *Load {
	return &Load{
		rctx:    rctx,
		root:    root,
		parent:  parent,
		tableID: tableID,
		branch:  branch,
		author:  author,
		message: message,
		meta:    meta,
	}
}

func (l *Load) Pull(done bool) (vector.Any, error) {
	if l.done {
		l.done = false
		return nil, nil
	}
	if done {
		if _, err := l.parent.Pull(true); err != nil {
			return nil, err
		}
		l.done = false
		return nil, nil
	}
	if len(l.branch) == 0 {
		l.branch = "main"
	}
	l.done = true
	table, err := l.root.OpenTable(l.rctx.Context, l.tableID)
	if err != nil {
		return nil, err
	}
	branch, err := table.OpenBranchByName(l.rctx.Context, l.branch)
	if err != nil {
		return nil, err
	}
	commitID, err := branch.Load(l.rctx.Context, l.rctx.Sctx, l.parent, l.author, l.message, l.meta)
	if err != nil {
		return nil, err
	}
	val := super.NewBytes(commitID[:])
	return sbuf.Dematerialize(l.rctx.Sctx, val), nil
}
