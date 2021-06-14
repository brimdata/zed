package del

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakecli"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
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
	zedapi.Cmd.Add(Delete)
}

type Command struct {
	lake   *zedlake.Command
	commit bool
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.BoolVar(&c.commit, "commit", false, "commit added data if successfully written")
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
	tags, err := lakecli.ParseIDs(args)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return errors.New("no data or commit tags specified")
	}
	if c.commit {
		if _, err := pool.Delete(ctx, tags, c.CommitRequest()); err != nil {
			return err
		}
		if !c.lake.Flags.Quiet() {
			fmt.Println("deletion successful")
		}
		return nil
	}
	commitID, err := pool.Delete(ctx, tags, nil)
	if err != nil {
		return err
	}
	if !c.lake.Flags.Quiet() {
		fmt.Printf("%s staged to delete\n", commitID)
	}
	return nil
}
