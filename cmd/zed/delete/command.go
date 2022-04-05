package del

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/segmentio/ksuid"
)

var Cmd = &charm.Spec{
	Name:  "delete",
	Usage: "delete id [id ...]",
	Short: "delete data objects from a pool branch",
	Long: `
The delete command takes a list of data object IDs and
deletes references to those object from HEAD by commiting a new
delete operation to HEAD.
Once the delete operation completes, the deleted data is no longer seen
when read data from the pool.

If the -where flag is specified, delete will remove all values that return true
the for the provided boolean expressions . Currently the filter expressions is
limited in that it must be against the pool key and it must be a single relative
comparison (e.g., -where 'ts <= now() - 3h').

No data is actually removed from the lake.  Instead, a delete
operation is an action in the pool's commit journal.  Any delete
can be "undone" by adding the commits back to the log using
"zed revert".
`,
	New: New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	lakeFlags lakeflags.Flags
	cli.CommitFlags
	where string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.CommitFlags.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	f.StringVar(&c.where, "where", "", "delete by pool key predicate")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolName := head.Pool
	if poolName == "" {
		return lakeflags.ErrNoHEAD
	}
	poolID, err := lake.PoolID(ctx, poolName)
	if err != nil {
		return err
	}
	var commit ksuid.KSUID
	if c.where != "" {
		if len(args) > 0 {
			return errors.New("too many arguments")
		}
		commit, err = c.deleteByPredicate(ctx, lake, head, poolID)
	} else {
		commit, err = c.deleteByIDs(ctx, lake, head, poolID, args)
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s delete committed\n", commit)
	}
	return nil
}

func (c *Command) deleteByIDs(ctx context.Context, lake api.Interface, head *lakeparse.Commitish, poolID ksuid.KSUID, args []string) (ksuid.KSUID, error) {
	ids, err := lakeparse.ParseIDs(args)
	if err != nil {
		return ksuid.Nil, err
	}
	if len(ids) == 0 {
		return ksuid.Nil, errors.New("no data object IDs specified")
	}
	return lake.Delete(ctx, poolID, head.Branch, ids, c.CommitMessage())
}

func (c *Command) deleteByPredicate(ctx context.Context, lake api.Interface, head *lakeparse.Commitish, poolID ksuid.KSUID) (ksuid.KSUID, error) {
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return ksuid.Nil, err
	}
	return lake.DeleteByPredicate(ctx, poolID, head.Branch, c.where, c.CommitMessage())
}
