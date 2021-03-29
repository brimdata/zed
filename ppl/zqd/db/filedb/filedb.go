package filedb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/pkg/iosrc"
	"github.com/brimdata/zq/ppl/zqd/auth"
	"github.com/brimdata/zq/ppl/zqd/db/schema"
	"github.com/brimdata/zq/zqe"
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

func (db *FileDB) load(ctx context.Context) (dbdata, error) {
	var data dbdata
	b, err := iosrc.ReadFile(ctx, db.path)
	if err != nil {
		return dbdata{}, err
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return dbdata{}, err
	}
	if data.Version != currentVersion {
		return dbdata{}, fmt.Errorf("expected db version %d, found %d", currentVersion, data.Version)
	}
	return data, nil
}

func (db *FileDB) save(ctx context.Context, data dbdata) error {
	data.Version = currentVersion
	return iosrc.Replace(ctx, db.path, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(data)
	})
}

func (db *FileDB) CreateSpace(ctx context.Context, row schema.SpaceRow) error {
	if row.ID == "" {
		return zqe.ErrInvalid("row must have an id")
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return err
	}

	for _, r := range dbdata.SpaceRows {
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

	dbdata.SpaceRows = append(dbdata.SpaceRows, row)
	return db.save(ctx, dbdata)
}

func (db *FileDB) GetSpace(ctx context.Context, id api.SpaceID) (schema.SpaceRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return schema.SpaceRow{}, err
	}
	for _, row := range dbdata.SpaceRows {
		if row.ID == id {
			return row, nil
		}
	}
	return schema.SpaceRow{}, zqe.ErrNotFound()
}

func (db *FileDB) ListSpaces(ctx context.Context, tenantID auth.TenantID) ([]schema.SpaceRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return nil, err
	}
	var rows []schema.SpaceRow
	for _, r := range dbdata.SpaceRows {
		if r.TenantID == tenantID {
			rows = append(rows, r)
		}
	}
	return rows, nil
}

func (db *FileDB) UpdateSpaceName(ctx context.Context, id api.SpaceID, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return err
	}

	idx := -1
	for i, r := range dbdata.SpaceRows {
		if r.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return zqe.ErrNotFound()
	}
	for _, r := range dbdata.SpaceRows {
		if r.TenantID != dbdata.SpaceRows[idx].TenantID {
			continue
		}
		if r.Name == name {
			return zqe.ErrConflict("space with name '%s' already exists", name)
		}
	}

	dbdata.SpaceRows[idx].Name = name
	return db.save(ctx, dbdata)
}

func (db *FileDB) DeleteSpace(ctx context.Context, id api.SpaceID) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return err
	}

	rows := dbdata.SpaceRows
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
	dbdata.SpaceRows = rows
	return db.save(ctx, dbdata)
}

func (db *FileDB) CreateIntake(ctx context.Context, in schema.IntakeRow) error {
	if in.ID == "" || in.TenantID == "" {
		return zqe.ErrInvalid("intake row must have an id and tenant_id")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return err
	}
	for _, r := range dbdata.IntakeRows {
		if in.ID == r.ID {
			return zqe.ErrExists()
		}
		if r.TenantID != in.TenantID {
			continue
		}
		if in.Name == r.Name {
			return zqe.ErrConflict("intake with name '%s' already exists", in.Name)
		}
	}
	dbdata.IntakeRows = append(dbdata.IntakeRows, in)
	return db.save(ctx, dbdata)
}

func (db *FileDB) GetIntake(ctx context.Context, id api.IntakeID) (schema.IntakeRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return schema.IntakeRow{}, err
	}
	for _, row := range dbdata.IntakeRows {
		if row.ID == id {
			return row, nil
		}
	}
	return schema.IntakeRow{}, zqe.ErrNotFound()
}

func (db *FileDB) ListIntakes(ctx context.Context, tenantID auth.TenantID) ([]schema.IntakeRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return nil, err
	}
	var rows []schema.IntakeRow
	for _, r := range dbdata.IntakeRows {
		if r.TenantID == tenantID {
			rows = append(rows, r)
		}
	}
	return rows, nil
}

func (db *FileDB) UpdateIntake(ctx context.Context, in schema.IntakeRow) error {
	if in.ID == "" || in.TenantID == "" {
		return zqe.ErrInvalid("intake row must have an id and tenant_id")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return err
	}
	rows := dbdata.IntakeRows
	idx := -1
	for i := range rows {
		if rows[i].ID == in.ID {
			idx = i
		}
	}
	if idx == -1 {
		return zqe.ErrNotFound()
	}
	for _, r := range dbdata.IntakeRows {
		if r.TenantID != in.TenantID {
			continue
		}
		if r.Name == in.Name {
			return zqe.ErrConflict("intake with name '%s' already exists", in.Name)
		}
	}
	dbdata.IntakeRows[idx] = in
	return db.save(ctx, dbdata)
}

func (db *FileDB) DeleteIntake(ctx context.Context, id api.IntakeID) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	dbdata, err := db.load(ctx)
	if err != nil {
		return err
	}
	rows := dbdata.IntakeRows
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
	dbdata.IntakeRows = rows
	return db.save(ctx, dbdata)
}
