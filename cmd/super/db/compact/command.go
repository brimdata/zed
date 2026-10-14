package compact

import (
	"flag"
	"fmt"

	"github.com/superdb/super/cli/commitflags"
	"github.com/superdb/super/cli/tableflags"
	"github.com/superdb/super/cmd/super/db"
	"github.com/superdb/super/dbid"
	"github.com/superdb/super/pkg/charm"
)

var spec = &charm.Spec{
	Name:  "compact",
	Usage: "compact id id [id ...]",
	Short: "compact data objects on a table branch",
	Long: `
See https://superdb.org/command/db.html#super-db-compact
`,
	New: New,
}

type Command struct {
	*db.Command
	commitFlags commitflags.Flags
	tableFlags  tableflags.Flags
}

func init() {
	db.Spec.Add(spec)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*db.Command)}
	c.commitFlags.SetFlags(f)
	c.tableFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	ids, err := dbid.ParseIDs(args)
	if err != nil {
		return err
	}
	db, err := c.DBFlags.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.tableFlags.HEAD()
	if err != nil {
		return err
	}
	tableID, err := db.TableID(ctx, head.Table)
	if err != nil {
		return err
	}
	commit, err := db.Compact(ctx, tableID, head.Branch, ids, c.commitFlags.CommitMessage())
	if err == nil && !c.DBFlags.Quiet {
		fmt.Printf("%s compaction committed\n", commit)
	}
	return err
}
