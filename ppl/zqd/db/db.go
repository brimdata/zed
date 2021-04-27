package db

import (
	"context"
	"flag"
	"fmt"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/ppl/zqd/auth"
	"github.com/brimdata/zed/ppl/zqd/db/filedb"
	"github.com/brimdata/zed/ppl/zqd/db/schema"
	"go.uber.org/zap"
)

type DB interface {
	CreateSpace(context.Context, schema.SpaceRow) error
	GetSpace(context.Context, api.SpaceID) (schema.SpaceRow, error)
	ListSpaces(context.Context, auth.TenantID) ([]schema.SpaceRow, error)
	UpdateSpaceName(context.Context, api.SpaceID, string) error
	DeleteSpace(context.Context, api.SpaceID) error

	CreateIntake(context.Context, schema.IntakeRow) error
	DeleteIntake(context.Context, api.IntakeID) error
	GetIntake(context.Context, api.IntakeID) (schema.IntakeRow, error)
	ListIntakes(context.Context, auth.TenantID) ([]schema.IntakeRow, error)
	UpdateIntake(context.Context, schema.IntakeRow) error
}

type DBKind string

// Set implements the flag.Value interface allowing DBKind to be used as a
// command line flag.
func (k *DBKind) Set(s string) error {
	switch s {
	case string(DBUnspecified), string(DBFile):
		*k = DBKind(s)
		return nil
	}
	return fmt.Errorf("unsupported db kind: %s", s)
}

// String implements the flag.Value interface allowing DBKind to be used as a
// command line flag.
func (k DBKind) String() string {
	if k == "" {
		k = DBFile
	}
	return string(k)
}

const (
	DBUnspecified DBKind = ""
	DBFile        DBKind = "file"
)

type Config struct {
	Kind DBKind
}

// Init is called after flags have been parsed.
func (d *Config) Init() error {
	return nil
}

func (d *Config) SetFlags(fs *flag.FlagSet) {
	fs.Var(&d.Kind, "db.kind", "the kind of database backing space data (values: file)")
}

func Open(ctx context.Context, logger *zap.Logger, conf Config, root iosrc.URI) (DB, error) {
	logger = logger.Named("database")
	switch conf.Kind {
	case DBFile, DBUnspecified:
		return filedb.Open(ctx, logger, root)

	default:
		return nil, fmt.Errorf("db.Open: unknown DBKind %q", conf.Kind)
	}
}
