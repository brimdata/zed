package apiserver

import (
	"context"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/postgres"
)

type DB interface {
	CreateSpace(context.Context, SpaceRow) error
	CreateSubspace(context.Context, SpaceRow) error
	GetSpace(context.Context, api.SpaceID) (SpaceRow, error)
	ListSpaces(context.Context) ([]SpaceRow, error)
	UpdateSpaceName(context.Context, api.SpaceID, string) error
	DeleteSpace(context.Context, api.SpaceID) error
}

type DBKind string

const (
	DBUnspecified DBKind = ""
	DBFile        DBKind = "file"
	DBPostgres    DBKind = "postgres"
)

type DBConfig struct {
	Kind     DBKind
	Postgres postgres.Config
}

type SpaceRow struct {
	tableName struct{}          `pg:"space"` // This is needed so the postgres orm knows the correct table name
	ID        api.SpaceID       `json:"id"`
	DataURI   iosrc.URI         `json:"data_uri"`
	Name      string            `json:"name"`
	ParentID  api.SpaceID       `json:"parent_id"`
	Storage   api.StorageConfig `json:"storage"`
}
