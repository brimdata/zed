package squash

import (
	"errors"
	"flag"
	"fmt"

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
}

type Command struct {
	*zedlake.Command
	lakeFlags zedlake.Flags
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	ids, err := zedlake.ParseIDs(args)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return errors.New("no commit tags specified")
	}
	commit, err := pool.Squash(ctx, ids, c.Date.Ts(), c.User, c.Message)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("squashed commit in staging: %s\n", commit)
	}
	return nil
}
