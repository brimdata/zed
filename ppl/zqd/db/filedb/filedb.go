package filedb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/auth"
	"github.com/brimsec/zq/ppl/zqd/db/schema"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

const dbname = "zqd.json"

type FileDB struct {
	mu     sync.Mutex
	logger *zap.Logger
	path   iosrc.URI
}

func Open(ctx context.Context, logger *zap.Logger, root iosrc.URI) (*FileDB, error) {
	if err := migrateOldConfig(ctx, logger, root); err != nil {
		logger.Error("Error migrating old multifile configuration", zap.Error(err))
		return nil, err
	}
	dburi := root.AppendPath(dbname)
	if err := migrateFileDatabase(ctx, dburi); err != nil {
		logger.Error("Error migrating database file", zap.Error(err))
		return nil, err
	}
	db := &FileDB{path: dburi, logger: logger}
	db.logger.Info("Loaded", zap.String("kind", "file"), zap.String("uri", dburi.String()))
	return db, nil
}

func (db *FileDB) load(ctx context.Context) ([]schema.SpaceRow, error) {
	b, err := iosrc.ReadFile(ctx, db.path)
	if err != nil {
		return nil, err
	}
	var data dbdata
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	if data.Version != currentVersion {
		return nil, fmt.Errorf("expected db version %d, found %d", currentVersion, data.Version)
	}
	return data.SpaceRows, nil
}

func (db *FileDB) save(ctx context.Context, lcs []schema.SpaceRow) error {
	return iosrc.Replace(ctx, db.path, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(dbdata{
			Version:   currentVersion,
			SpaceRows: lcs,
		})
	})
}

func (db *FileDB) CreateSpace(ctx context.Context, row schema.SpaceRow) error {
	if row.ID == "" {
		return zqe.ErrInvalid("row must have an id")
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	rows, err := db.load(ctx)
	if err != nil {
		return err
	}

	for _, r := range rows {
		if row.ID == r.ID {
			return zqe.ErrExists()
		}
		if r.TenantID != row.TenantID {
			continue
		}
		if row.Name == r.Name {
			return zqe.ErrConflict("space with name '%s' already exists", row.Name)
		}
	}

	return db.save(ctx, append(rows, row))
}

func (db *FileDB) GetSpace(ctx context.Context, id api.SpaceID) (schema.SpaceRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	rows, err := db.load(ctx)
	if err != nil {
		return schema.SpaceRow{}, err
	}
	for i := range rows {
		if rows[i].ID == id {
			return rows[i], nil
		}
	}
	return schema.SpaceRow{}, zqe.ErrNotFound()
}

func (db *FileDB) ListSpaces(ctx context.Context, tenantID auth.TenantID) ([]schema.SpaceRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	all, err := db.load(ctx)
	if err != nil {
		return nil, err
	}
	var rows []schema.SpaceRow
	for _, r := range all {
		if r.TenantID == tenantID {
			rows = append(rows, r)
		}
	}
	return rows, nil
}

func (db *FileDB) UpdateSpaceName(ctx context.Context, id api.SpaceID, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	rows, err := db.load(ctx)
	if err != nil {
		return err
	}

	idx := -1
	for i, r := range rows {
		if r.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return zqe.ErrNotFound()
	}
	for _, r := range rows {
		if r.TenantID != rows[idx].TenantID {
			continue
		}
		if r.Name == name {
			return zqe.ErrConflict("space with name '%s' already exists", name)
		}
	}

	rows[idx].Name = name
	return db.save(ctx, rows)
}

func (db *FileDB) DeleteSpace(ctx context.Context, id api.SpaceID) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	rows, err := db.load(ctx)
	if err != nil {
		return err
	}

	idx := -1
	for i := range rows {
		if rows[i].ID == id {
			idx = i
		}
	}
	if idx == -1 {
		return zqe.ErrNotFound()
	}

	rows[idx] = rows[len(rows)-1]
	rows = rows[:len(rows)-1]
	return db.save(ctx, rows)
}
