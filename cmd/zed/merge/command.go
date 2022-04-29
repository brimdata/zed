package merge

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "merge",
	Usage: "merge branch",
	Short: "merge current branch into another",
	Long: `
`,
	New: New,
}

type Command struct {
	*root.Command
	force       bool
	commitFlags cli.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
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
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.LakeFlags.HEAD()
	if err != nil {
		return err
	}
	if head.Pool == "" {
		return lakeflags.ErrNoHEAD
	}
	if head.Branch == "" || targetBranch == "" {
		return errors.New("both a child and a parent branch name must be specified")
	}
	if head.Branch == "main" && !c.force {
		return errors.New("merging the main branch into another branch is unusual; use -f to force")
	}
	poolID, err := lake.PoolID(ctx, head.Pool)
	if err != nil {
		return err
	}
	if _, err = lake.MergeBranch(ctx, poolID, head.Branch, targetBranch, c.commitFlags.CommitMessage()); err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		fmt.Printf("%q: merged into branch %q\n", head.Branch, targetBranch)
	}
	return nil
}
