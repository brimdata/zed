package postgresdb

import (
	"context"
	"errors"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/ppl/zqd/auth"
	"github.com/brimsec/zq/ppl/zqd/db/schema"
	"github.com/brimsec/zq/zqe"
	"github.com/go-pg/pg/v10"
	"go.uber.org/zap"
)

type PostgresDB struct {
	db     *pg.DB
	logger *zap.Logger
}

func Open(ctx context.Context, logger *zap.Logger, conf Config) (*PostgresDB, error) {
	db := pg.Connect(&conf.Options)
	if err := db.Ping(ctx); err != nil {
		return nil, err
	}
	logger.Info("Connected", zap.String("kind", "postgres"), zap.String("uri", conf.StringRedacted()))
	return &PostgresDB{db, logger}, nil
}

func (d *PostgresDB) CreateSpace(ctx context.Context, row schema.SpaceRow) error {
	if row.ID == "" {
		return zqe.ErrInvalid("row must have an id")
	}

	_, err := d.db.ModelContext(ctx, &row).Insert()
	if IsUniqueViolation(err) {
		return zqe.ErrConflict("space with name '%s' already exists", row.Name)
	}
	return err
}

func (d *PostgresDB) GetSpace(ctx context.Context, id api.SpaceID) (schema.SpaceRow, error) {
	var space schema.SpaceRow
	_, err := d.db.QueryOneContext(ctx, &space, "SELECT * FROM space WHERE id = ?", id)
	if errors.Is(err, pg.ErrNoRows) {
		err = zqe.ErrNotFound("subspace parent not found")
	}
	return space, err
}

func (d *PostgresDB) ListSpaces(ctx context.Context, tenantID auth.TenantID) ([]schema.SpaceRow, error) {
	var spaces []schema.SpaceRow
	_, err := d.db.QueryContext(ctx, &spaces, "SELECT * FROM space WHERE tenant_id = ?", tenantID)
	return spaces, err
}

func (d *PostgresDB) UpdateSpaceName(ctx context.Context, id api.SpaceID, name string) error {
	_, err := d.db.ExecContext(ctx, "UPDATE space SET name = ? WHERE id = ?", name, id)
	if errors.Is(err, pg.ErrNoRows) {
		return zqe.ErrNotFound()
	} else if IsUniqueViolation(err) {
		return zqe.ErrConflict("space with name '%s' already exists", name)
	}
	return err
}

func (d *PostgresDB) DeleteSpace(ctx context.Context, id api.SpaceID) error {
	_, err := d.db.ExecOneContext(ctx, "DELETE FROM space WHERE id = ?", id)
	if IsForeignKeyViolation(err) {
		return zqe.ErrConflict("cannot delete space with subspaces")
	}
	return err
}
