package branch

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/pkg/charm"
)

var Branch = &charm.Spec{
	Name:  "branch",
	Usage: "branch -p pool[@commit] branch",
	Short: "create a new branch",
	Long: `
The lake branch command creates a new branch with the indicated name.
The branch's parent is given by the -p option, where the "@commit"
suffix to the pool may be either another branch name or a commit ID.
The pool argument may be either a pool name or a pool ID.

If a branch commit ID or branch name is not provided, then the branch
will be made from the current tip of "main".

If the -d option is specified, then the branch is deleted.  No data is
deleted by this operation and the deleted branch can be easily recreated by
running the branch command again with the commit ID desired.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Branch)
	zedapi.Cmd.Add(Branch)
}

type Command struct {
	lake      zedlake.Command
	delete    bool
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.delete, "d", false, "delete the branch instead of creating it")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("a new branch name must be given")
	}
	branchName := args[0]
	poolName, commitName := c.lakeFlags.Branch()
	if poolName == "" {
		return errors.New("a branch/commit must be specified with -p")
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolID, err := parser.ParseID(poolName)
	if err != nil {
		poolID, err = lake.PoolID(ctx, poolName)
		if err != nil {
			return err
		}
	}
	parentCommit, err := parser.ParseID(commitName)
	if err != nil {
		parentCommit, err = lake.CommitObject(ctx, poolID, commitName)
		if err != nil {
			return err
		}
	}
	if c.delete {
		if err := lake.RemoveBranch(ctx, poolID, branchName); err != nil {
			return err
		}
		if !c.lakeFlags.Quiet {
			fmt.Printf("branch deleted: %s\n", branchName)
		}
		return nil
	}
	if err := lake.CreateBranch(ctx, poolID, branchName, parentCommit); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%q: branch created\n", branchName)
	}
	return nil
}
