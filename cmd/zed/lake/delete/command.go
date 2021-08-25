package del

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/pkg/charm"
)

var Delete = &charm.Spec{
	Name:  "delete",
	Usage: "delete -p pool[/branch] id [id ...]",
	Short: "delete commits or data objects from a pool branch",
	Long: `
"zed lake delete" takes a list of commit tags and/or data object tags
in the specified pool branch and deletes all of the corresponding data objects.
Once the delete operation completes, the deleted data is no longer seen
when read data from the pool.

No data is actually removed from the lake.  Instead, a delete
operation is an action in the pool's commit journal.  Any delete
can be "undone" by adding the commits back to the log using
"zed lake add".
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
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.commit, "commit", false, "commit added data if successfully written")
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
	poolName, branchName := c.lakeFlags.Branch()
	if poolName == "" {
		return errors.New("name of pool must be supplied with -p option")
	}
	poolID, branchID, err := lake.IDs(ctx, poolName, branchName)
	if err != nil {
		return err
	}
	tags, err := parser.ParseIDs(args)
	if err != nil {
		return err
	}
	if len(tags) == 0 {
		return errors.New("no data or commit tags specified")
	}
	commit, err := lake.Delete(ctx, poolID, branchID, tags, c.CommitRequest())
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s delete committed\n", commit)
	}
	return nil
}
