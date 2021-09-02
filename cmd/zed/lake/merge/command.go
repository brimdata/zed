package merge

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Merge = &charm.Spec{
	Name:  "merge",
	Usage: "merge -p pool@child-branch parent-branch",
	Short: "merge a branch into another",
	Long: `
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Merge)
	zedapi.Cmd.Add(Merge)
}

type Command struct {
	lake        zedlake.Command
	force       bool
	lakeFlags   lakeflags.Flags
	commitFlags zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.force, "f", false, "force merge of main into a child")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("parent branch must be given")
	} else if len(args) > 1 {
		return errors.New("too many arguments")
	}
	parentBranch := args[0]
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolName, childBranch := c.lakeFlags.Names()
	if poolName == "" {
		return errors.New("pool name must be specified with -p")
	}
	if childBranch == "" || parentBranch == "" {
		return errors.New("both a child and a parent branch name must be specified")
	}
	if childBranch == "main" && !c.force {
		return errors.New("merging the main branch into another branch is unusual; use -f to force")
	}
	poolID, err := lake.PoolID(ctx, poolName)
	if err != nil {
		return err
	}
	if _, err = lake.MergeBranch(ctx, poolID, childBranch, parentBranch, c.commitFlags.CommitMessage()); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%q: merged into branch %q\n", childBranch, parentBranch)
	}
	return nil
}
