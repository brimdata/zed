package branch

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/outputflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
)

var Branch = &charm.Spec{
	Name:  "branch",
	Usage: "branch new-branch [base]",
	Short: "create a new branch",
	Long: `
The lake branch command creates a new branch with the indicated name.
If specified, base is either an existing branch name or a commit ID
and provides the new branch's base.  If not specified, then HEAD is assumed.

The branch command does not checkout the new branch.

If the -d option is specified, then the branch is deleted.  No data is
deleted by this operation and the deleted branch can be easily recreated by
running the branch command again with the commit ID desired.

If no branch is currently checked out, then "-HEAD pool@base" can be
used to specify the desired pool for the new branch.

`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Branch)
	zedapi.Cmd.Add(Branch)
}

type Command struct {
	lake        zedlake.Command
	delete      bool
	lakeFlags   lakeflags.Flags
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.delete, "d", false, "delete the branch instead of creating it")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return c.list(ctx, lake)
	}
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	branchName := args[0]
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolName := head.Pool
	if poolName == "" {
		return errors.New("a pool name must be included: pool@branch")
	}
	poolID, err := lakeparse.ParseID(poolName)
	if err != nil {
		poolID, err = lake.PoolID(ctx, poolName)
		if err != nil {
			return err
		}
	}
	parentCommit, err := lakeparse.ParseID(head.Branch)
	if err != nil {
		parentCommit, err = lake.CommitObject(ctx, poolID, head.Branch)
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

func (c *Command) list(ctx context.Context, lake api.Interface) error {
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolName := head.Pool
	if poolName == "" {
		return errors.New("must be on a checked out out branch to list the branches in the same pool")
	}
	query := fmt.Sprintf("from '%s':branches", poolName)
	if c.outputFlags.Format == "lake" {
		c.outputFlags.WriterOpts.Lake.Head = head.Branch
	}
	zw, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	_, err = lake.Query(ctx, driver.NewCLI(zw), nil, query)
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	return err
}
