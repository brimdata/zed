package use

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

var Use = &charm.Spec{
	Name:  "use",
	Usage: "use [-p pool] [-b] branch [base]",
	Short: "use a branch",
	Long: `
The lake use command sets the working branch as indicated.
This allows commands like load, rebase, merge etc to function without
having to specify the working branch.  The branch specifier may also be
a commit ID, in which case you entered a headless state and commands
like load that require a branch name for HEAD will report an error.

The use command is like "git checkuout" but there is no local copy of
the lake data.  Rather, the local HEAD state links invocations of
lake commands run locally or through zapi directly to the remote lake.

Use may also be run with -p to indicate a pool name.  In this case,
the main branch of the specified pool is checked out.

Any command that relies upon HEAD can also be run with the -use option
to refer to a different HEAD without executing an explicit "use" command.
The HEAD option has the form "pool@branch" where pool is the name or ID of an
existing pool and branch is the name of the branch or a commit ID.
While the use of HEAD is convenient for interactive CLI sessions,
automation and orchestration tools are better of hard-wiring the
HEAD references in each lake command via -use.

If the -b option is provided, then a new branch with the indicated
name is created with base HEAD and checked out.

The use command merely checks that the branch exists and updates the
file ~/.zed_head.  This file simply contains a pointer to the HEAD branch
and thus provides the default for the -use option.  This way, multiple working
directories can contain different HEAD pointers (along with your local files)
and you can easily switch between windows without having to continually
re-specify a new HEAD.  Unlike Git, all the commited pool data remains
in the lake and is not copied to this local directory.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Use)
	zedapi.Cmd.Add(Use)
}

type Command struct {
	lake      zedlake.Command
	branch    bool
	poolName  string
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.branch, "b", false, "create the branch then check it out")
	f.StringVar(&c.poolName, "p", "", "check out the main branch of the given pool")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	var branchName, baseName string
	switch len(args) {
	case 0:
		if c.poolName == "" {
			return errors.New("a branch name or commit ID must be given")
		}
	case 1:
		branchName = args[0]
	case 2:
		branchName = args[0]
		baseName = args[1]
	default:
		return errors.New("too many arguments")
	}
	if baseName != "" && !c.branch {
		return errors.New("cannot specify a base for a new branch without -b")
	}
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolName := head.Pool
	if c.poolName != "" {
		poolName = c.poolName
		if branchName == "" {
			branchName = "main"
		}
	}
	commitish, err := lakeparse.ParseCommitish(branchName)
	if err != nil {
		return err
	}
	poolSpec, branchSpec := commitish.Pool, commitish.Branch
	if poolSpec != "" {
		poolName, branchName = poolSpec, branchSpec
	}
	if poolName == "" {
		if c.poolName == "" {
			return lakeflags.ErrNoHEAD
		}

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
		if baseName == "" {
			baseName = head.Branch
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
		if c.poolName != "" {
			fmt.Printf("Switched to %sbranch %q on pool %q\n", new, branchName, c.poolName)
		} else {
			fmt.Printf("Switched to %sbranch %q\n", new, branchName)
		}
	}
	return nil
}
