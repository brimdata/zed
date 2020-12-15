package apiserver

import (
	"context"
	"errors"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/ppl/zqd/postgres"
	"github.com/brimsec/zq/zqe"
	"github.com/go-pg/pg/v10"
)

type PostgresDB struct {
	db *pg.DB
}

func OpenPostgresDB(ctx context.Context, conf postgres.Config) (*PostgresDB, error) {
	db := pg.Connect(&conf.Options)
	if err := db.Ping(ctx); err != nil {
		return nil, err
	}
	return &PostgresDB{db}, nil
}

func (d *PostgresDB) CreateSpace(ctx context.Context, row SpaceRow) error {
	if row.ID == "" {
		return zqe.ErrInvalid("row must have an id")
	}
	if row.ParentID != "" {
		return zqe.ErrInvalid("parent id cannot be set for non-subspace spaces")
	}

	_, err := d.db.ModelContext(ctx, &row).Insert()
	if postgres.IsUniqueViolation(err) {
		return zqe.ErrConflict("space with name '%s' already exists", row.Name)
	}
	return err
}

func (d *PostgresDB) CreateSubspace(ctx context.Context, row SpaceRow) error {
	if row.ParentID == "" {
		return zqe.ErrInvalid("subspace must have parent id")
	}

	err := d.CreateSpace(ctx, row)
	if postgres.IsForeignKeyViolation(err) {
		return zqe.ErrNotFound("subspace parent not found")
	}
	return err
}

func (d *PostgresDB) GetSpace(ctx context.Context, id api.SpaceID) (SpaceRow, error) {
	var space SpaceRow
	_, err := d.db.QueryOneContext(ctx, &space, "SELECT * FROM space WHERE id = ?", id)
	if errors.Is(err, pg.ErrNoRows) {
		err = zqe.ErrNotFound("subspace parent not found")
	}
	return space, err
}

func (d *PostgresDB) ListSpaces(ctx context.Context) ([]SpaceRow, error) {
	var spaces []SpaceRow
	_, err := d.db.QueryContext(ctx, &spaces, "SELECT * FROM space")
	return spaces, err
}

func (d *PostgresDB) UpdateSpaceName(ctx context.Context, id api.SpaceID, name string) error {
	_, err := d.db.ExecContext(ctx, "UPDATE space SET name = ? WHERE id = ?", name, id)
	if errors.Is(err, pg.ErrNoRows) {
		return zqe.ErrNotFound()
	} else if postgres.IsUniqueViolation(err) {
		return zqe.ErrConflict("space with name '%s' already exists", name)
	}
	return err
}

func (d *PostgresDB) DeleteSpace(ctx context.Context, id api.SpaceID) error {
	_, err := d.db.ExecOneContext(ctx, "DELETE FROM space WHERE id = ?", id)
	if postgres.IsForeignKeyViolation(err) {
		return zqe.ErrConflict("cannot delete space with subspaces")
	}
	return err
}
