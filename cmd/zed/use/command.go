package use

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "use",
	Usage: "use [pool][@branch]",
	Short: "use a branch",
	Long: `
The use command prints or sets the working pool and branch.  Setting these
values allows commands like load, rebase, merge, etc. to function without
having to specify the working branch.  The branch specifier may also be
a commit ID, in which case you enter a headless state and commands
like load that require a branch will report an error.

The use command is like "git checkuout" but there is no local copy of
the lake data.  Rather, the local HEAD state influences commands as
they access the lake.

With no argument, use prints the working pool and branch.

With an argument of the form "pool", use sets the working pool as indicated
and the working branch to "main".

With an argument of the form "pool@branch", use sets the working pool and
branch as indicated.

With an argument of the form "@branch", use sets only the working branch.
The working pool must already be set.

The pool must be the name or ID of an existing pool.  The branch must be
the name of an existing branch or a commit ID.

Any command that relies upon HEAD can also be run with the -use option
to refer to a different HEAD without executing an explicit "use" command.
While the use of HEAD is convenient for interactive CLI sessions,
automation and orchestration tools are better off hard-wiring the
HEAD references in each lake command via -use.

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

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*root.Command)}, nil
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
		head, err := c.LakeFlags.HEAD()
		if err != nil {
			return errors.New("default pool and branch unset")
		}
		fmt.Printf("HEAD at %s\n", head)
		return nil
	}
	commitish, err := lakeparse.ParseCommitish(args[0])
	if err != nil {
		return err
	}
	if commitish.Pool == "" {
		head, err := c.LakeFlags.HEAD()
		if err != nil {
			return errors.New("default pool unset")
		}
		commitish.Pool = head.Pool
	}
	if commitish.Branch == "" {
		commitish.Branch = "main"
	}
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	poolID, err := lakeparse.ParseID(commitish.Pool)
	if err != nil {
		poolID, err = lake.PoolID(ctx, commitish.Pool)
		if err != nil {
			return err
		}
	}
	if _, err = lake.CommitObject(ctx, poolID, commitish.Branch); err != nil {
		return err
	}
	if err := lakeflags.WriteHead(commitish.Pool, commitish.Branch); err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		fmt.Printf("Switched to branch %q on pool %q\n", commitish.Branch, commitish.Pool)
	}
	return nil
}
