package rename

import (
	"errors"
	"flag"
	"fmt"

	"github.com/superdb/super/cmd/super/db"
	"github.com/superdb/super/pkg/charm"
)

var spec = &charm.Spec{
	Name:  "rename",
	Usage: "rename old-name new-name",
	Short: "rename a table",
	Long: `
See https://superdb.org/command/db.html#super-db-rename
`,
	New: New,
}

func init() {
	db.Spec.Add(spec)
}

type Command struct {
	*db.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*db.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 2 {
		return errors.New("two tables must be provided")
	}
	oldName := args[0]
	newName := args[1]
	db, err := c.DBFlags.Open(ctx)
	if err != nil {
		return err
	}
	tableID, err := db.TableID(ctx, oldName)
	if err != nil {
		return err
	}
	if err := db.RenameTable(ctx, tableID, newName); err != nil {
		return err
	}
	if !c.DBFlags.Quiet {
		fmt.Printf("table %s renamed from %s to %s\n", tableID, oldName, newName)
	}
	return nil
}
