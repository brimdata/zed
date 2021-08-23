package del

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
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

If the -staging flag is set, the specified commit(s) will be
deleted from staging. If the specified commit(s) do not exist,
an error will be returned.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Delete)
	zedapi.Cmd.Add(Delete)
}

type Command struct {
	lake      zedlake.Command
	commit    bool
	lakeFlags lakeflags.Flags
	staging   bool
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.commit, "commit", false, "commit deleted data if successfully written")
	f.BoolVar(&c.staging, "staging", false, "perform delete on staged commits")
	c.CommitFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	pool, err := lake.LookupPool(ctx, c.lakeFlags.PoolName)
	if err != nil {
		return err
	}
	tags, err := lakeflags.ParseIDs(args)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return errors.New("no data or commit tags specified")
	}
	if c.staging {
		if c.commit {
			return errors.New("cannot have both -staging and -commit flags activated")
		}
		for _, id := range tags {
			if err := lake.DeleteFromStaging(ctx, pool.ID, id); err != nil {
				return err
			}
			fmt.Printf("%s deleted from staging\n", id.String())
		}
		return nil
	}
	if c.commit {
		if _, err := lake.Delete(ctx, pool.ID, tags, c.CommitRequest()); err != nil {
			return err
		}
		if !c.lakeFlags.Quiet {
			fmt.Println("deletion successful")
		}
		return nil
	}
	commitID, err := lake.Delete(ctx, pool.ID, tags, nil)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s staged to delete\n", commitID)
	}
	return nil
}
