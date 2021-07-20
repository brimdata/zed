package commit

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
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
	zedapi.Cmd.Add(Commit)
}

type Command struct {
	lake      zedlake.Command
	lakeFlags lakeflags.Flags
	user      string
	message   string
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("zed lake commit: at least one pending commit tag must be specified")
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	pool, err := lake.LookupPoolByName(ctx, c.lakeFlags.PoolName)
	if err != nil {
		return err
	}
	ids, err := lakeflags.ParseIDs(args)
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
		commitID, err = lake.Squash(ctx, pool.ID, ids)
		if err != nil {
			return err
		}
	}
	if err := lake.Commit(ctx, commitID, pool.ID, *c.CommitRequest()); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s committed\n", commitID)
	}
	return nil
}
