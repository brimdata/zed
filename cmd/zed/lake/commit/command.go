package commit

import (
	"errors"
	"flag"
	"fmt"
	"os"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/signalctx"
)

var Commit = &charm.Spec{
	Name:  "commit",
	Usage: "commit [options] tag [tag ...]",
	Short: "transactionally commit data from staging into data pool",
	Long: `
The commit command...
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
		return errors.New("zed lake add: at least one input file must be specified (- for stdin)")
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
	if len(ids) == 0 {
		return errors.New("no commit tags specified")
	}
	if len(ids) > 1 {
		return errors.New("issue #2543: squash on commit not yet implemented")
	}
	if err := pool.Commit(ctx, ids[0], c.Date.Ts(), c.User, c.Message); err != nil {
		return err
	}
	fmt.Println("commit successful")
	return nil
}
