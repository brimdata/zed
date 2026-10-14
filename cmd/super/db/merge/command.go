package merge

import (
	"errors"
	"flag"
	"fmt"

	"github.com/superdb/super/cli/commitflags"
	"github.com/superdb/super/cli/dbflags"
	"github.com/superdb/super/cli/tableflags"
	"github.com/superdb/super/cmd/super/db"
	"github.com/superdb/super/pkg/charm"
)

var spec = &charm.Spec{
	Name:  "merge",
	Usage: "merge branch",
	Short: "merge current branch into another",
	Long: `
See https://superdb.org/command/db.html#super-db-merge
`,
	New: New,
}

func init() {
	db.Spec.Add(spec)
}

type Command struct {
	*db.Command
	commitFlags commitflags.Flags
	tableFlags  tableflags.Flags
	force       bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*db.Command)}
	c.commitFlags.SetFlags(f)
	c.tableFlags.SetFlags(f)
	f.BoolVar(&c.force, "f", false, "force merge of main into a target")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("merge target branch must be given")
	} else if len(args) > 1 {
		return errors.New("too many arguments")
	}
	targetBranch := args[0]
	db, err := c.DBFlags.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.tableFlags.HEAD()
	if err != nil {
		return err
	}
	if head.Table == "" {
		return dbflags.ErrNoHEAD
	}
	if head.Branch == "" || targetBranch == "" {
		return errors.New("both a child and a parent branch name must be specified")
	}
	if head.Branch == "main" && !c.force {
		return errors.New("merging the main branch into another branch is unusual; use -f to force")
	}
	tableID, err := db.TableID(ctx, head.Table)
	if err != nil {
		return err
	}
	if _, err = db.MergeBranch(ctx, tableID, head.Branch, targetBranch, c.commitFlags.CommitMessage()); err != nil {
		return err
	}
	if !c.DBFlags.Quiet {
		fmt.Printf("%q: merged into branch %q\n", head.Branch, targetBranch)
	}
	return nil
}
