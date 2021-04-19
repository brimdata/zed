package drop

import (
	"errors"
	"flag"
	"fmt"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

//XXX TBD: add drop by pool ID

var Cmd = &charm.Spec{
	Name:  "drop",
	Usage: "drop -p name",
	Short: "delete a data pool from a lake",
	Long: `
"zed lake drop" removes the named pool from the lake.
The -p flag must be given.

DANGER ZONE.
There is no prompting or second chances here so use carefully.
Once the pool is delted, its data is gone.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Cmd)
}

type Command struct {
	lake *zedlake.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{lake: parent.(*zedlake.Command)}, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	name := c.lake.Flags.PoolName
	if name == "" {
		return errors.New("name of pool must be supplied with -p option")
	}
	lk, err := c.lake.Flags.Open(ctx)
	if err != nil {
		return err
	}
	if err := lk.RemovePool(ctx, name); err != nil {
		return err
	}
	if !c.lake.Flags.Quiet {
		fmt.Printf("pool deleted: %s\n", name)
	}
	return nil
}
