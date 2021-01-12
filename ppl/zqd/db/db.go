package db

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/db/filedb"
	"github.com/brimsec/zq/ppl/zqd/db/postgresdb"
	"github.com/brimsec/zq/ppl/zqd/db/schema"
	"go.uber.org/zap"
)

type DB interface {
	CreateSpace(context.Context, schema.SpaceRow) error
	CreateSubspace(context.Context, schema.SpaceRow) error
	GetSpace(context.Context, api.SpaceID) (schema.SpaceRow, error)
	ListSpaces(context.Context) ([]schema.SpaceRow, error)
	UpdateSpaceName(context.Context, api.SpaceID, string) error
	DeleteSpace(context.Context, api.SpaceID) error
}

type DBKind string

const (
	DBUnspecified DBKind = ""
	DBFile        DBKind = "file"
	DBPostgres    DBKind = "postgres"
)

type Config struct {
	Kind     DBKind
	Postgres postgresdb.Config
}

func Open(ctx context.Context, logger *zap.Logger, conf Config, root iosrc.URI) (DB, error) {
	logger = logger.Named("db")
	switch conf.Kind {
	case DBFile, DBUnspecified:
		return filedb.Open(ctx, logger, root)

	case DBPostgres:
		return postgresdb.Open(ctx, logger, conf.Postgres)

	default:
		return nil, fmt.Errorf("db.Open: unknown DBKind %q", conf.Kind)
	}
}
