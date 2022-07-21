package vector

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/commitflags"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
)

var add = &charm.Spec{
	Name:  "add",
	Usage: "add [options] id [id, ]",
	Short: "create vectorized forms of one or more data objects",
	Long: `
The vector add command creates vector forms of one or more data objects specified
by the indicated object IDs.
`,
	New: newAdd,
}

type addCommand struct {
	*Command
	commitFlags commitflags.Flags
}

func newAdd(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &addCommand{Command: parent.(*Command)}
	c.commitFlags.SetFlags(f)
	return c, nil
}

func (c *addCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	ids, err := lakeparse.ParseIDs(args)
	if err != nil {
		return err
	}
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.LakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolID, err := lake.PoolID(ctx, head.Pool)
	if err != nil {
		return err
	}
	commit, err := lake.AddVectors(ctx, poolID, head.Branch, ids, c.commitFlags.CommitMessage())
	if err == nil && !c.LakeFlags.Quiet {
		fmt.Printf("%s vectors added\n", commit)
	}
	return err
}
