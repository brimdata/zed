package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqe"
)

type FileDB struct {
	mu   sync.Mutex
	path iosrc.URI
}

func CreateFileDB(ctx context.Context, path iosrc.URI, rows []SpaceRow) (*FileDB, error) {
	db := &FileDB{path: path}
	if err := db.save(ctx, rows); err != nil {
		return nil, err
	}
	return db, nil
}

func OpenFileDB(ctx context.Context, path iosrc.URI) (*FileDB, error) {
	db := &FileDB{path: path}
	// Verify file exists & is readable.
	if _, err := db.load(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

const dbversion = 4

type dbdataV4 struct {
	Version   int        `json:"version"`
	SpaceRows []SpaceRow `json:"space_rows"`
}

func (db *FileDB) load(ctx context.Context) ([]SpaceRow, error) {
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

func (db *FileDB) save(ctx context.Context, lcs []SpaceRow) error {
	return iosrc.Replace(ctx, db.path, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(dbdataV4{
			Version:   dbversion,
			SpaceRows: lcs,
		})
	})
}

func (db *FileDB) CreateSpace(ctx context.Context, row SpaceRow) error {
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

func (db *FileDB) CreateSubspace(ctx context.Context, row SpaceRow) error {
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

func (db *FileDB) GetSpace(ctx context.Context, id api.SpaceID) (SpaceRow, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	rows, err := db.load(ctx)
	if err != nil {
		return SpaceRow{}, err
	}
	for i := range rows {
		if rows[i].ID == id {
			return rows[i], nil
		}
	}
	return SpaceRow{}, zqe.ErrNotFound()
}

func (db *FileDB) ListSpaces(ctx context.Context) ([]SpaceRow, error) {
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
