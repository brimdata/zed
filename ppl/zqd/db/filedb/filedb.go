package filedb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/db/filedb/oldconfig"
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

func Create(ctx context.Context, logger *zap.Logger, path iosrc.URI, rows []schema.SpaceRow) (*FileDB, error) {
	db := &FileDB{path: path, logger: logger}
	if err := db.save(ctx, rows); err != nil {
		return nil, err
	}
	db.logger.Info("Created")
	return db, nil
}

func Open(ctx context.Context, logger *zap.Logger, root iosrc.URI) (*FileDB, error) {
	dburi := root.AppendPath(dbname)
	logger = logger.With(
		zap.String("kind", "file"),
		zap.String("uri", dburi.String()),
	)
	exists, err := iosrc.Exists(ctx, dburi)
	if err != nil {
		return nil, err
	}
	if exists {
		return open(ctx, logger, dburi)
	}

	// Since the dbfile doesn't exist, we check if we need to migrate the older
	// per-space config files into a new dbfile.
	configs, err := oldconfig.LoadConfigs(ctx, logger, root)
	if err != nil {
		return nil, err
	}
	var rows []schema.SpaceRow
	for id, config := range configs {
		datauri := config.DataURI
		if datauri.IsZero() {
			datauri = root.AppendPath(string(id))
		}
		rows = append(rows, schema.SpaceRow{
			ID:      id,
			Name:    config.Name,
			DataURI: datauri,
			Storage: config.Storage,
		})
		for _, subcfg := range config.Subspaces {
			openopts := subcfg.OpenOptions
			rows = append(rows, schema.SpaceRow{
				ID:       subcfg.ID,
				ParentID: id,
				Name:     subcfg.Name,
				DataURI:  datauri,
				Storage: api.StorageConfig{
					Kind: api.ArchiveStore,
					Archive: &api.ArchiveConfig{
						OpenOptions: &openopts,
					},
				},
			})
		}
	}
	return Create(ctx, logger, dburi, rows)
}

func open(ctx context.Context, logger *zap.Logger, path iosrc.URI) (*FileDB, error) {
	db := &FileDB{path: path, logger: logger}
	// Verify file exists & is readable.
	if _, err := db.load(ctx); err != nil {
		return nil, err
	}
	db.logger.Info("Loaded")
	return db, nil
}

const dbversion = 4

type dbdataV4 struct {
	Version   int               `json:"version"`
	SpaceRows []schema.SpaceRow `json:"space_rows"`
}

func (db *FileDB) load(ctx context.Context) ([]schema.SpaceRow, error) {
	b, err := iosrc.ReadFile(ctx, db.path)
	if err != nil {
		return nil, err
	}
	var data dbdataV4
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	if data.Version != dbversion {
		return nil, fmt.Errorf("expected db version %d, found %d", dbversion, data.Version)
	}
	return data.SpaceRows, nil
}

func (db *FileDB) save(ctx context.Context, lcs []schema.SpaceRow) error {
	return iosrc.Replace(ctx, db.path, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(dbdataV4{
			Version:   dbversion,
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
		if row.Name == r.Name {
			return zqe.ErrConflict("space with name '%s' already exists", row.Name)
		}
		if row.ID == r.ID {
			return zqe.ErrExists()
		}
	}

	return db.save(ctx, append(rows, row))
}

func (db *FileDB) CreateSubspace(ctx context.Context, row schema.SpaceRow) error {
	if row.ID == "" {
		return zqe.ErrInvalid("row must have an id")
	}
	if row.ParentID == "" {
		return zqe.ErrInvalid("subspace must have parent id")
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	rows, err := db.load(ctx)
	if err != nil {
		return err
	}

	parentIdx := -1
	for i, r := range rows {
		if row.Name == r.Name {
			return zqe.ErrConflict("space with name '%s' already exists", row.Name)
		}
		if row.ID == r.ID {
			return zqe.ErrExists()
		}
		if row.ParentID == r.ID {
			parentIdx = i
		}
	}
	if parentIdx == -1 {
		return zqe.ErrNotFound("subspace parent not found")
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

func (db *FileDB) ListSpaces(ctx context.Context) ([]schema.SpaceRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.load(ctx)
}

func (db *FileDB) UpdateSpaceName(ctx context.Context, id api.SpaceID, name string) error {
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
			continue
		}
		if rows[i].Name == name {
			return zqe.ErrConflict("space with name '%s' already exists", name)
		}
	}
	if idx == -1 {
		return zqe.ErrNotFound()
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
		if rows[i].ParentID == id {
			return zqe.E(zqe.Conflict, "cannot delete space with subspaces")
		}
	}
	if idx == -1 {
		return zqe.ErrNotFound()
	}

	rows[idx] = rows[len(rows)-1]
	rows = rows[:len(rows)-1]
	return db.save(ctx, rows)
}
