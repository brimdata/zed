package apiserver

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqe"
)

type FileDb struct {
	mu   sync.Mutex
	path iosrc.URI
}

func CreateFileDb(ctx context.Context, path iosrc.URI, rows []SpaceRow) (*FileDb, error) {
	ldb := &FileDb{path: path}
	if err := ldb.save(ctx, rows); err != nil {
		return nil, err
	}
	return ldb, nil
}

func OpenFileDb(ctx context.Context, path iosrc.URI) (*FileDb, error) {
	ldb := &FileDb{path: path}
	// Verify file exists & is readable.
	if _, err := ldb.load(ctx); err != nil {
		return nil, err
	}
	return ldb, nil
}

const dbversion = 4

type dbdataV4 struct {
	Version   int        `json:"version"`
	SpaceRows []SpaceRow `json:"space_rows"`
}

func (ldb *FileDb) load(ctx context.Context) ([]SpaceRow, error) {
	b, err := iosrc.ReadFile(ctx, ldb.path)
	if err != nil {
		return nil, err
	}
	var lf dbdataV4
	if err := json.Unmarshal(b, &lf); err != nil {
		return nil, err
	}
	return lf.SpaceRows, nil
}

func (ldb *FileDb) save(ctx context.Context, lcs []SpaceRow) error {
	return iosrc.Replace(ctx, ldb.path, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(dbdataV4{
			Version:   dbversion,
			SpaceRows: lcs,
		})
	})
}

type SpaceRow struct {
	ID       api.SpaceID       `json:"id"`
	DataURI  iosrc.URI         `json:"data_uri"`
	Name     string            `json:"name"`
	ParentID api.SpaceID       `json:"parent_id"`
	Storage  api.StorageConfig `json:"storage"`
}

func (ldb *FileDb) CreateSpace(ctx context.Context, lr SpaceRow) error {
	if lr.ID == "" {
		return zqe.ErrInvalid("lr must have an id")
	}

	ldb.mu.Lock()
	defer ldb.mu.Unlock()
	rows, err := ldb.load(ctx)
	if err != nil {
		return err
	}

	for _, l := range rows {
		if lr.Name == l.Name {
			return zqe.ErrConflict("space with name '%s' already exists", lr.Name)
		}
		if lr.ID == l.ID {
			return zqe.ErrExists()
		}
	}

	return ldb.save(ctx, append(rows, lr))
}

func (ldb *FileDb) CreateSubspace(ctx context.Context, lr SpaceRow) error {
	if lr.ID == "" {
		return zqe.ErrInvalid("lr must have an id")
	}
	if lr.ParentID == "" {
		return zqe.ErrInvalid("subspace must have parent id")
	}

	ldb.mu.Lock()
	defer ldb.mu.Unlock()
	rows, err := ldb.load(ctx)
	if err != nil {
		return err
	}

	parentIdx := -1
	for i, l := range rows {
		if lr.Name == l.Name {
			return zqe.ErrConflict("space with name '%s' already exists", lr.Name)
		}
		if lr.ID == l.ID {
			return zqe.ErrExists()
		}
		if lr.ParentID == l.ID {
			parentIdx = i
		}
	}
	if parentIdx == -1 {
		return zqe.ErrNotFound("subspace parent not found")
	}

	return ldb.save(ctx, append(rows, lr))
}

func (ldb *FileDb) GetSpace(ctx context.Context, id api.SpaceID) (SpaceRow, error) {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()
	rows, err := ldb.load(ctx)
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

func (ldb *FileDb) ListSpaces(ctx context.Context) ([]SpaceRow, error) {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()
	return ldb.load(ctx)
}

func (ldb *FileDb) UpdateSpaceName(ctx context.Context, id api.SpaceID, name string) error {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()
	rows, err := ldb.load(ctx)
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
	return ldb.save(ctx, rows)
}

func (ldb *FileDb) DeleteSpace(ctx context.Context, id api.SpaceID) error {
	ldb.mu.Lock()
	defer ldb.mu.Unlock()
	rows, err := ldb.load(ctx)
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
	return ldb.save(ctx, rows)
}
