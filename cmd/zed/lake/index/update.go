package index

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
)

var update = &charm.Spec{
	Name:  "update",
	Usage: "update",
	Short: "index all object in a branch using the current set of index rules",
	New:   newUpdate,
}

type updateCommand struct {
	*Command
	zedlake.CommitFlags
}

func newUpdate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &updateCommand{Command: parent.(*Command)}
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *updateCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.lakeFlags.HEAD()
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
	commit, err := lake.UpdateIndex(ctx, poolID, head.Branch)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s committed\n", commit)
	}
	return nil
}
