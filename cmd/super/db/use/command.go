package use

import (
	"errors"
	"flag"
	"fmt"

	"github.com/superdb/super/cli/tableflags"
	"github.com/superdb/super/cmd/super/db"
	"github.com/superdb/super/dbid"
	"github.com/superdb/super/pkg/charm"
)

//XXX should use be called connect?

var spec = &charm.Spec{
	Name:  "use",
	Usage: "use [table][@branch]",
	Short: "use a branch or print current branch and database",
	Long: `
See https://superdb.org/command/db.html#super-db-use
`,
	New: New,
}

func init() {
	db.Spec.Add(spec)
}

type Command struct {
	*db.Command
	tableFlags tableflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*db.Command)}
	c.tableFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	if len(args) == 0 {
		head, err := c.tableFlags.HEAD()
		if err != nil {
			return errors.New("default table and branch unset")
		}
		fmt.Printf("HEAD at %s\n", head)
		if u, err := c.DBFlags.URI(); err == nil {
			fmt.Printf("Attached to %s\n", u)
		}
		return nil
	}
	committish, err := dbid.ParseCommittish(args[0])
	if err != nil {
		return err
	}
	if committish.Table == "" {
		head, err := c.tableFlags.HEAD()
		if err != nil {
			return errors.New("default table unset")
		}
		committish.Table = head.Table
	}
	if committish.Branch == "" {
		committish.Branch = "main"
	}
	db, err := c.DBFlags.Open(ctx)
	if err != nil {
		return err
	}
	tableID, err := db.TableID(ctx, committish.Table)
	if err != nil {
		return err
	}
	if _, err = db.CommitObject(ctx, tableID, committish.Branch); err != nil {
		return err
	}
	if err := tableflags.WriteHead(committish.Table, committish.Branch); err != nil {
		return err
	}
	if !c.DBFlags.Quiet {
		fmt.Printf("Switched to branch %q on table %q\n", committish.Branch, committish.Table)
	}
	return nil
}
