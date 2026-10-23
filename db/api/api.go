package api

import (
	"context"
	"fmt"
	"strings"
	"uuid"

	"github.com/superdb/super"
	"github.com/superdb/super/api"
	"github.com/superdb/super/api/client"
	"github.com/superdb/super/compiler/srcfiles"
	"github.com/superdb/super/db"
	"github.com/superdb/super/db/commits"
	"github.com/superdb/super/db/tables"
	"github.com/superdb/super/dbid"
	"github.com/superdb/super/order"
	"github.com/superdb/super/pkg/nano"
	"github.com/superdb/super/sbuf"
	"github.com/superdb/super/sio"
	"github.com/superdb/super/vector"
	"github.com/superdb/super/vector/vio"
	"go.uber.org/zap"
)

type Interface interface {
	Root() *db.Root
	Query(ctx context.Context, query []srcfiles.Input) (vio.Scanner, error)
	TableID(ctx context.Context, tableName string) (uuid.UUID, error)
	CommitObject(ctx context.Context, tableID uuid.UUID, branchName string) (uuid.UUID, error)
	CreateTable(context.Context, string, order.SortKeys, uint64, uint64) (uuid.UUID, error)
	RemoveTable(context.Context, uuid.UUID) error
	RenameTable(context.Context, uuid.UUID, string) error
	CreateBranch(ctx context.Context, tableID uuid.UUID, name string, parent uuid.UUID) error
	RemoveBranch(ctx context.Context, tableID uuid.UUID, branchName string) error
	MergeBranch(ctx context.Context, tableID uuid.UUID, childBranch, parentBranch string, message api.CommitMessage) (uuid.UUID, error)
	Compact(ctx context.Context, tableID uuid.UUID, branch string, objects []uuid.UUID, message api.CommitMessage) (uuid.UUID, error)
	Load(ctx context.Context, sctx *super.Context, tableID uuid.UUID, branch string, r vio.Puller, message api.CommitMessage) (uuid.UUID, error)
	Delete(ctx context.Context, tableID uuid.UUID, branchName string, tags []uuid.UUID, message api.CommitMessage) (uuid.UUID, error)
	DeleteWhere(ctx context.Context, tableID uuid.UUID, branchName, src string, commit api.CommitMessage) (uuid.UUID, error)
	Revert(ctx context.Context, tableID uuid.UUID, branch string, commitID uuid.UUID, commit api.CommitMessage) (uuid.UUID, error)
	Vacate(ctx context.Context, table string, time nano.Ts, dryrun bool) ([]uuid.UUID, error)
	Vacuum(ctx context.Context, table, revision string, dryrun bool) ([]uuid.UUID, error)
}

func Connect(ctx context.Context, logger *zap.Logger, u string) (Interface, error) {
	if IsRemote(u) {
		return NewRemoteDB(client.NewConnectionTo(u)), nil
	}
	return OpenLocalDB(ctx, logger, u)
}

func IsRemote(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

func LookupTableByName(ctx context.Context, api Interface, name string) (*tables.Config, error) {
	b := newBuffer(tables.Config{})
	query := fmt.Sprintf("from :tables | name == '%s'", name)
	q, err := api.Query(ctx, srcfiles.Plain(query))
	if err != nil {
		return nil, err
	}
	defer q.Pull(true)
	if err := vio.Copy(b, q); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%q: table not found", name)
	case 1:
		table, ok := b.results[0].(*tables.Config)
		if !ok {
			return nil, fmt.Errorf("internal error: table record has wrong type: %T", b.results[0])
		}
		return table, nil
	default:
		return nil, fmt.Errorf("internal error: multiple tables found with same name: %s", name)
	}
}

func GetTables(ctx context.Context, api Interface) ([]*tables.Config, error) {
	b := newBuffer(tables.Config{})
	q, err := api.Query(ctx, srcfiles.Plain("from :tables"))
	if err != nil {
		return nil, err
	}
	defer q.Pull(true)
	if err := vio.Copy(b, q); err != nil {
		return nil, err
	}
	var pls []*tables.Config
	for _, r := range b.results {
		pls = append(pls, r.(*tables.Config))
	}
	return pls, nil
}

func LookupTableByID(ctx context.Context, api Interface, id uuid.UUID) (*tables.Config, error) {
	b := newBuffer(tables.Config{})
	query := fmt.Sprintf("from :tables | id == %s", id)
	q, err := api.Query(ctx, srcfiles.Plain(query))
	if err != nil {
		return nil, err
	}
	defer q.Pull(true)
	if err := vio.Copy(b, q); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%s: table not found", id)
	case 1:
		table, ok := b.results[0].(*tables.Config)
		if !ok {
			return nil, fmt.Errorf("internal error: table record has wrong type: %T", b.results[0])
		}
		return table, nil
	default:
		return nil, fmt.Errorf("internal error: multiple tabled found with same id: %s", id)
	}
}

func LookupBranchByName(ctx context.Context, api Interface, tableName, branchName string) (*db.BranchMeta, error) {
	b := newBuffer(db.BranchMeta{})
	query := fmt.Sprintf("from :branches | table.name == '%s' branch.name == '%s'", tableName, branchName)
	q, err := api.Query(ctx, srcfiles.Plain(query))
	if err != nil {
		return nil, err
	}
	defer q.Pull(true)
	if err := vio.Copy(b, q); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%q: branch not found", tableName+"/"+branchName)
	case 1:
		branch, ok := b.results[0].(*db.BranchMeta)
		if !ok {
			return nil, fmt.Errorf("internal error: branch record has wrong type: %T", b.results[0])
		}
		return branch, nil
	default:
		return nil, fmt.Errorf("internal error: multiple branches found with same name: %s", tableName+"/"+branchName)
	}
}

func LookupBranchByID(ctx context.Context, api Interface, id uuid.UUID) (*db.BranchMeta, error) {
	b := newBuffer(db.BranchMeta{})
	query := fmt.Sprintf("from :branches | branch.id == %s", id)
	q, err := api.Query(ctx, srcfiles.Plain(query))
	if err != nil {
		return nil, err
	}
	defer q.Pull(true)
	if err := vio.Copy(b, q); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%s: branch not found", id)
	case 1:
		branch, ok := b.results[0].(*db.BranchMeta)
		if !ok {
			return nil, fmt.Errorf("internal error: branch record has wrong type: %T", b.results[0])
		}
		return branch, nil
	default:
		return nil, fmt.Errorf("internal error: multiple branches found with same id: %s", id)
	}
}

func GetCommit(ctx context.Context, api Interface, tableName, revision string) (*commits.Commit, error) {
	tableID, err := api.TableID(ctx, tableName)
	if err != nil {
		return nil, err
	}
	commit, err := dbid.ParseID(revision)
	if err != nil {
		// LookupBranchByName(ctx
		if commit, err = api.CommitObject(ctx, tableID, revision); err != nil {
			return nil, err
		}
	}
	b := newBuffer(commits.Commit{})
	// The log holds actions of several types and only commits have an id, so
	// skip the others rather than have where report them as side errors.
	query := fmt.Sprintf("from %q:log | where not is_error(id) and id == %s", tableID, commit)
	q, err := api.Query(ctx, srcfiles.Plain(query))
	if err != nil {
		return nil, err
	}
	defer q.Pull(true)
	if err := vio.Copy(b, q); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%s: commit not found", commit)
	case 1:
		commit, ok := b.results[0].(*commits.Commit)
		if !ok {
			return nil, fmt.Errorf("internal error: branch record has wrong type: %T", b.results[0])
		}
		return commit, nil
	default:
		return nil, fmt.Errorf("internal error: multiple commits found with same id: %s", commit)
	}
}

type buffer struct {
	unmarshaler *super.Unmarshaler
	results     []any
}

var _ sio.Writer = (*buffer)(nil)

func newBuffer(types ...any) *buffer {
	u := super.NewUnmarshaler()
	u.Bind(types...)
	u.SetContext(super.NewContext())
	return &buffer{unmarshaler: u}
}

func (b *buffer) Push(vec vector.Any) error {
	return sbuf.WriteVec(b, vec)
}

func (b *buffer) Write(val super.Value) error {
	var v any
	if err := b.unmarshaler.Unmarshal(val, &v); err != nil {
		return err
	}
	b.results = append(b.results, v)
	return nil
}
