package testdb

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/cmd/pgctl/root"
	"github.com/brimsec/zq/ppl/zqd/db/postgresdb"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mccanne/charm"
)

var Migrate = &charm.Spec{
	Name:  "migrate",
	Usage: "migrate [-m migration dir] [postgres options]",
	Short: "perform a schema migration on a database",
	New:   New,
}

type Command struct {
	*root.Command
	postgres   postgresdb.Config
	migrations string
}

func init() {
	root.CLI.Add(Migrate)
}

func New(parent charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	fs.StringVar(&c.migrations, "m", "", "migrations directory")
	c.postgres.SetFlags(fs)
	return c, nil
}

func (c *Command) Run(args []string) error {
	if err := c.Init(&c.postgres); err != nil {
		return err
	}
	if c.migrations == "" {
		return errors.New("argument '-m' (migrations directory) is required")
	}

	u, err := iosrc.ParseURI(c.migrations)
	if err != nil {
		return err
	}

	m, err := migrate.New(u.String(), c.postgres.String())
	if err != nil {
		return err
	}

	oldversion, _, err := m.Version()
	if err != nil && errors.Is(err, migrate.ErrNilVersion) {
		return err
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Printf("database already to update to date: %d\n", oldversion)
			return nil
		}
		return err
	}

	version, _, err := m.Version()
	if err != nil {
		return fmt.Errorf("error migrated current version: %w", err)
	}

	fmt.Printf("db %q successfully migrated: from version %d to version %d\n", c.postgres.Database, oldversion, version)
	return nil
}
