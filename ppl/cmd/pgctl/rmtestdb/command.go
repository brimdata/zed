package rmtestdb

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/ppl/cmd/pgctl/root"
	"github.com/brimdata/zed/ppl/zqd/db/postgresdb"
	"github.com/go-pg/pg/v10"
)

var RmTestDB = &charm.Spec{
	Name:  "rmtestdb",
	Usage: "testdb [-p postgres url] testdb_name",
	Short: "remove db created with testdb",
	New:   New,
}

type Command struct {
	charm.Command
	postgres postgresdb.Config
}

func init() {
	root.CLI.Add(RmTestDB)
}

func New(parent charm.Command, fs *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	fs.Var(&c.postgres, "p", "postgres url")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if c.postgres.IsEmpty() {
		return errors.New("argument '-p' (postgres url) is required")
	}
	if len(args) == 0 {
		return errors.New("must provide test db name as argument")
	}
	testdb := args[0]
	if !strings.HasPrefix(testdb, "test_") {
		return fmt.Errorf("the provided test db name is not a test db (must start with 'test_')")
	}

	db := pg.Connect(&c.postgres.Options)
	if err := db.Ping(context.TODO()); err != nil {
		return err
	}

	_, err := db.Exec("DROP DATABASE ?", pg.Ident(testdb))
	return err
}
