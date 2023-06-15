package index

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/poolflags"
	"github.com/brimdata/zed/pkg/charm"
)

var update = &charm.Spec{
	Name:  "update",
	Usage: "update [rule ...]",
	Short: "index all unindexed data objects",
	Long: `
The index update command creates index objects for all data objects that don't have an
index object for the provided list of index rules.

If no rules are given, the update is performed for all index rules.`,
	New: newUpdate,
}

type updateCommand struct {
	*Command
	poolFlags poolflags.Flags
}

func newUpdate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &updateCommand{Command: parent.(*Command)}
	c.poolFlags.SetFlags(f)
	return c, nil
}

func (c *updateCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.poolFlags.HEAD()
	if err != nil {
		return err
	}
	if head.Pool == "" {
		return lakeflags.ErrNoHEAD
	}
	poolID, err := lake.PoolID(ctx, head.Pool)
	if err != nil {
		return err
	}
	commit, err := lake.UpdateIndex(ctx, args, poolID, head.Branch)
	if err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		fmt.Printf("%s committed\n", commit)
	}
	return nil
}
