package testdb

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/cmd/pgctl/root"
	"github.com/brimsec/zq/ppl/zqd/db/postgresdb"
	"github.com/go-pg/pg/v10"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/mccanne/charm"
	"github.com/segmentio/ksuid"
)

var TestDB = &charm.Spec{
	Name:  "testdb",
	Usage: "testdb [-p postgres url] [-m migration dir] [options]",
	Short: "create a test db with up-to-date migrations",
	New:   New,
}

type Command struct {
	charm.Command
	postgres   postgresdb.Config
	migrations string
}

func init() {
	root.CLI.Add(TestDB)
}

func New(parent charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	fs.Var(&c.postgres, "p", "postgres url")
	fs.StringVar(&c.migrations, "m", "", "migrations directory")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if c.postgres.IsEmpty() {
		return errors.New("argument '-p' (postgres url) is required")
	}
	if c.migrations == "" {
		return errors.New("argument '-m' (migrations directory) is required")
	}

	u, err := iosrc.ParseURI(c.migrations)
	if err != nil {
		return err
	}

	db := pg.Connect(&c.postgres.Options)
	if err := db.Ping(context.TODO()); err != nil {
		return err
	}

	dbname := "test_" + ksuid.New().String()
	if _, err := db.Exec("CREATE DATABASE ?;", pg.Ident(dbname)); err != nil {
		return err
	}
	c.postgres.Database = dbname

	m, err := migrate.New(u.String(), c.postgres.String())
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil {
		return err
	}

	fmt.Println(dbname)
	return nil
}
