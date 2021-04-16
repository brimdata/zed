package del

import (
	"errors"
	"flag"
	"fmt"
	"os"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/signalctx"
)

var Delete = &charm.Spec{
	Name:  "delete",
	Usage: "delete id [id ...]",
	Short: "delete commits or segments from a data pool",
	Long: `
"zed lake delete" takes a list of commit tags or data segment tags
in the specified pool
and stages a deletion commit for each object listed and
each object in the listed commits.

Once the delete is comitted, the deleted data is no longer seen
when read data from the pool.

No data is actually removed from the lake.  Instead, a delete
operation is an action in the pool's commit journal.  Any delete
can be "undone" by adding the commits back to the log using
"zed lake add".

It is an error to delete commits or objects that are not
visible in the lake.  The staged deletes will be checked for
consistency, but the final decision on a consistency is made
when the staged delete commit is actually committed,
e.g., with "zed lake commit".

To delete commits in staging, use the "zed lake stage" command.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Delete)
}

type Command struct {
	*zedlake.Command
	commit    bool
	lakeFlags zedlake.Flags
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.BoolVar(&c.commit, "commit", false, "commit added data if successfully written")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	tags, err := zedlake.ParseIDs(args)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return errors.New("no data or commit tags specified")
	}
	ids, err := pool.LookupTags(ctx, tags)
	if err != nil {
		return err
	}
	commitID, err := pool.Delete(ctx, ids)
	if err != nil {
		return err
	}
	if c.commit {
		if err := pool.Commit(ctx, commitID, c.Date.Ts(), c.User, c.Message); err != nil {
			return err
		}
		if !c.lakeFlags.Quiet {
			fmt.Println("deletion successful")
		}
		return nil
	}
	if !c.lakeFlags.Quiet {
		txn, err := pool.LoadFromStaging(ctx, commitID)
		if err != nil {
			return err
		}
		fmt.Printf("commit %s in staging:\n", commitID)
		for _, action := range txn.Actions {
			if del, ok := action.(*actions.Delete); ok {
				fmt.Printf(" delete segment %s\n", del.ID)
			}
		}
	}
	return nil
}
