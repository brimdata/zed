package drop

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/superdb/super/cmd/super/db"
	"github.com/superdb/super/pkg/charm"
)

var spec = &charm.Spec{
	Name:  "drop",
	Usage: "drop table",
	Short: "delete a table from a database",
	Long: `
See https://superdb.org/command/db.html#super-db-drop
`,
	New: New,
}

func init() {
	db.Spec.Add(spec)
}

type Command struct {
	*db.Command
	force bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*db.Command)}
	f.BoolVar(&c.force, "f", false, "do not prompt for confirmation")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("a single table name must be specified")
	}
	db, err := c.DBFlags.Open(ctx)
	if err != nil {
		return err
	}
	tableName := args[0]
	tableID, err := db.TableID(ctx, tableName)
	if err != nil {
		return err
	}
	if err := c.confirm(tableName); err != nil {
		return err
	}
	if err := db.RemoveTable(ctx, tableID); err != nil {
		return err
	}
	if !c.DBFlags.Quiet {
		fmt.Printf("table deleted: %s\n", tableName)
	}
	return nil
}

func (c *Command) confirm(name string) error {
	if c.force {
		return nil
	}
	fmt.Printf("Are you sure you want to delete table %q? There is no going back... [y|n]\n", name)
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return err
	}
	input = strings.ToLower(input)
	if input == "y" || input == "yes" {
		return nil
	}
	return errors.New("operation canceled")
}
