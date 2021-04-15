package commit

import (
	"errors"
	"flag"
	"fmt"
	"os"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/signalctx"
	"github.com/segmentio/ksuid"
)

var Commit = &charm.Spec{
	Name:  "commit",
	Usage: "commit [options] tag [tag ...]",
	Short: "transactionally commit data from staging into pool",
	Long: `
The commit command takes one or more pending commits in a pool's staging area
and transactionally commits them to the pool.  If a write conflict
occurs (e.g., because a pending commit deletes data that no longer exists),
the commit is aborted and an error reported.

If multiple commit tags are specified, they are combined into a single new
commit as if a "zed lake squash" command were executed prior to the commit.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Commit)
}

type Command struct {
	*zedlake.Command
	user      string
	message   string
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
	defer c.Cleanup()
	if len(args) == 0 {
		return errors.New("zed lake commit: at least one pending commit tag must be specified")
	}
	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	ids, err := zedlake.ParseIDs(args)
	if err != nil {
		return err
	}
	var commitID ksuid.KSUID
	switch len(ids) {
	case 0:
		return errors.New("no commit tags specified")
	case 1:
		commitID = ids[0]
	default:
		commitID, err = pool.Squash(ctx, ids, c.Date.Ts(), c.User, c.Message)
		if err != nil {
			return err
		}
	}
	if err := pool.Commit(ctx, commitID, c.Date.Ts(), c.User, c.Message); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Println("commit successful")
	}
	return nil
}
