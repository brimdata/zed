package squash

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakecli"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Squash = &charm.Spec{
	Name:  "squash",
	Usage: "squash [options] tag [tag ...]",
	Short: "combine commits in a pool's staging area",
	Long: `
The squash command takes multiple pending commits in a pool
and combines them into a single pending commit, printing to stdout
the new tag of the squashed commits.  The combined commit can be
subsequently committed with "zed lake commit".

The order of the tags is significant as the pending commits are
assembled into a snapshot reflecting the indicated order
of any underlying add/delete operations.  If a delete operation
encounters a tag that is not present in the implied commit,
the squash will fail.  This integrity check is performed with
respect to the head of the pool's commit journal at the time it is run.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Squash)
	zedapi.Cmd.Add(Squash)
}

type Command struct {
	lake *zedlake.Command
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	pool, err := c.lake.Flags.OpenPool(ctx)
	if err != nil {
		return err
	}
	ids, err := lakecli.ParseIDs(args)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errors.New("no commit tags specified")
	}
	commit, err := pool.Squash(ctx, ids)
	if err != nil {
		return err
	}
	if !c.lake.Flags.Quiet() {
		fmt.Printf("squashed commit in staging: %s\n", commit)
	}
	return nil
}
