package api

import (
	"context"
	"errors"
	"io"
	"uuid"

	"github.com/superdb/super"
	"github.com/superdb/super/api"
	"github.com/superdb/super/api/client"
	"github.com/superdb/super/api/queryio"
	"github.com/superdb/super/compiler/srcfiles"
	"github.com/superdb/super/db"
	"github.com/superdb/super/dbid"
	"github.com/superdb/super/order"
	"github.com/superdb/super/pkg/nano"
	"github.com/superdb/super/sio"
	"github.com/superdb/super/sio/bsupio"
	"github.com/superdb/super/vector/vio"
)

type remote struct {
	conn *client.Connection
}

var _ Interface = (*remote)(nil)

func NewRemoteDB(conn *client.Connection) Interface {
	return &remote{conn}
}

func (l *remote) Root() *db.Root {
	return nil
}

func (r *remote) TableID(ctx context.Context, tableName string) (uuid.UUID, error) {
	if id, err := dbid.ParseID(tableName); err == nil {
		if _, err := LookupTableByID(ctx, r, id); err == nil {
			return id, nil
		}
	}
	config, err := LookupTableByName(ctx, r, tableName)
	if err != nil {
		return uuid.Nil(), err
	}
	return config.ID, nil
}

func (r *remote) CommitObject(ctx context.Context, tableID uuid.UUID, branchName string) (uuid.UUID, error) {
	res, err := r.conn.BranchGet(ctx, tableID, branchName)
	return res.Commit, err
}

func (r *remote) CreateTable(ctx context.Context, name string, sortKeys order.SortKeys, objectCap, frameCap uint64) (uuid.UUID, error) {
	res, err := r.conn.CreateTable(ctx, api.TablePostRequest{
		Name:      name,
		SortKeys:  sortKeys,
		ObjectCap: objectCap,
		FrameCap:  frameCap,
	})
	if err != nil {
		return uuid.Nil(), err
	}
	return res.Table.ID, err
}

func (r *remote) CreateBranch(ctx context.Context, tableID uuid.UUID, name string, at uuid.UUID) error {
	_, err := r.conn.CreateBranch(ctx, tableID, api.BranchPostRequest{
		Name:   name,
		Commit: at.String(),
	})
	return err
}

func (r *remote) RemoveBranch(ctx context.Context, tableID uuid.UUID, branchName string) error {
	return errors.New("TBD remote.RemoveBranch")
}

func (r *remote) MergeBranch(ctx context.Context, tableID uuid.UUID, childBranch, parentBranch string, message api.CommitMessage) (uuid.UUID, error) {
	res, err := r.conn.MergeBranch(ctx, tableID, childBranch, parentBranch, message)
	return res.Commit, err
}

func (r *remote) Compact(ctx context.Context, tableID uuid.UUID, branch string, objects []uuid.UUID, commit api.CommitMessage) (uuid.UUID, error) {
	res, err := r.conn.Compact(ctx, tableID, branch, objects, commit)
	return res.Commit, err
}

func (r *remote) RemoveTable(ctx context.Context, tableID uuid.UUID) error {
	return r.conn.RemoveTable(ctx, tableID)
}

func (r *remote) RenameTable(ctx context.Context, tableID uuid.UUID, name string) error {
	if name == "" {
		return errors.New("no table name provided")
	}
	return r.conn.RenameTable(ctx, tableID, api.TablePutRequest{Name: name})
}

func (r *remote) Load(ctx context.Context, _ *super.Context, tableID uuid.UUID, branchName string, reader vio.Puller, commit api.CommitMessage) (uuid.UUID, error) {
	pr, pw := io.Pipe()
	go func() {
		w := bsupio.NewColumnWriter(sio.NopCloser(pw))
		err := vio.Copy(w, reader)
		if err2 := w.Close(); err == nil {
			err = err2
		}
		pw.CloseWithError(err)
	}()
	res, err := r.conn.Load(ctx, tableID, branchName, api.MediaTypeBSUP, pr, commit)
	return res.Commit, err
}

func (r *remote) Revert(ctx context.Context, tableID uuid.UUID, branchName string, commitID uuid.UUID, message api.CommitMessage) (uuid.UUID, error) {
	res, err := r.conn.Revert(ctx, tableID, branchName, commitID, message)
	return res.Commit, err
}

func (r *remote) Query(ctx context.Context, inputs []srcfiles.Input) (vio.Scanner, error) {
	res, err := r.conn.Query(ctx, inputs)
	if err != nil {
		return nil, err
	}
	return queryio.NewScanner(ctx, res.Body)
}

func (r *remote) Delete(ctx context.Context, tableID uuid.UUID, branchName string, tags []uuid.UUID, commit api.CommitMessage) (uuid.UUID, error) {
	res, err := r.conn.Delete(ctx, tableID, branchName, tags, commit)
	return res.Commit, err
}

func (r *remote) DeleteWhere(ctx context.Context, tableID uuid.UUID, branchName, src string, commit api.CommitMessage) (uuid.UUID, error) {
	res, err := r.conn.DeleteWhere(ctx, tableID, branchName, src, commit)
	return res.Commit, err
}

func (r *remote) Vacate(ctx context.Context, table string, ts nano.Ts, dryrun bool) ([]uuid.UUID, error) {
	res, err := r.conn.Vacate(ctx, table, ts, dryrun)
	return res.CommitIDs, err
}

func (r *remote) Vacuum(ctx context.Context, table, revision string, dryrun bool) ([]uuid.UUID, error) {
	res, err := r.conn.Vacuum(ctx, table, revision, dryrun)
	return res.ObjectIDs, err
}
