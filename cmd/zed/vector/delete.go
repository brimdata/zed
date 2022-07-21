package vector

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/commitflags"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
)

var del = &charm.Spec{
	Name:  "delete",
	Usage: "delete [options] id [id, ]",
	Short: "deleted vectors from one or more data objects",
	Long: `
The vector delete command deletes vectors from of one or more data objects specified
by the indicated object IDs.  The references to the vectors is simply deleted
in the commit history.  The vacate command may be used to delete the actual data.
`,
	New: newDelete,
}

type deleteCommand struct {
	*Command
	commitFlags commitflags.Flags
}

func newDelete(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &deleteCommand{Command: parent.(*Command)}
	c.commitFlags.SetFlags(f)
	return c, nil
}

func (c *deleteCommand) Run(args []string) error {
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
	commit, err := lake.DeleteVectors(ctx, poolID, head.Branch, ids, c.commitFlags.CommitMessage())
	if err == nil && !c.LakeFlags.Quiet {
		fmt.Printf("%s vectors deleted\n", commit)
	}
	return err
}
