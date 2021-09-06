package branch

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
)

var Checkout = &charm.Spec{
	Name:  "checkout",
	Usage: "checkout [-b] branch",
	Short: "checkout a branch",
	Long: `
The lake checkout command sets the working branch as indicated.
This allows commands like load, rebase, merge etc to function without
having to specify the working branch.  The branch specifier may also be
a commit ID, in which case you entered a headless state and commands
like load that require a branch name for HEAD will report an error.

Any command that relies upon HEAD can also be run with the -HEAD option
to refer to a different HEAD without performing a checkout.
The HEAD option has the form "pool@branch" where pool is the name or ID of an
existing pool and branch is the name of the branch or a commit ID.
While checkouts of HEAD are useful for interactive CLI sessions,
automation and orchestration tools are better of hard-wiring the
HEAD references in each lake command using -HEAD.

If the -b option is provided, then a new branch with the indicated
name is created with base HEAD and checked out.

The checkout command merely checks that the branch exists and updates the
file ~/.zed_head.  This file simply contains a pointer to the HEAD branch
and thus provides the default for the -HEAD option.  This way, multiple working
directories can contain different HEAD pointers (along with your local files)
and you can easily switch between windows without having to continually
re-specify a new HEAD.  Unlike Git, all the commited pool data remains
in the lake and is not copied to this local directory.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Checkout)
	zedapi.Cmd.Add(Checkout)
}

type Command struct {
	lake      zedlake.Command
	branch    bool
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.branch, "b", false, "create the branch then check it out")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	switch len(args) {
	case 0:
		return errors.New("a branch name or commit ID must be given")
	case 1, 2:
	default:
		return errors.New("too many arguments")
	}
	branchName := args[0]
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolName, baseName := head.Pool, head.Branch
	commitish, err := lakeparse.ParseCommitish(branchName)
	if err != nil {
		return err
	}
	poolSpec, branchSpec := commitish.Pool, commitish.Branch
	if poolSpec != "" {
		poolName, branchName = poolSpec, branchSpec
	}
	if poolName == "" {
		return lakeflags.ErrNoHEAD
	}
	if len(args) == 2 {
		baseName = args[1]
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolID, err := lakeparse.ParseID(poolName)
	if err != nil {
		poolID, err = lake.PoolID(ctx, poolName)
		if err != nil {
			return err
		}
	}
	if c.branch {
		if _, err := lakeparse.ParseID(branchName); err == nil {
			return errors.New("new branch name cannot be a commit ID")
		}
		baseCommit, err := lakeparse.ParseID(baseName)
		if err != nil {
			baseCommit, err = lake.CommitObject(ctx, poolID, baseName)
			if err != nil {
				return err
			}
		}
		if err := lake.CreateBranch(ctx, poolID, branchName, baseCommit); err != nil {
			return err
		}
	} else if _, err = lake.CommitObject(ctx, poolID, branchName); err != nil {
		return err
	}
	if err := lakeflags.WriteHead(poolName, branchName); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		new := ""
		if c.branch {
			new = "a new "
		}
		fmt.Printf("Switched to %sbranch %q\n", new, branchName)
	}
	return nil
}
