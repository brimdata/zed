package drop

import (
	"context"
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
	*zedlake.Command
	lakeFlags zedlake.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	name := c.lakeFlags.PoolName
	if name == "" {
		return errors.New("name of pool must be supplied with -p option")
	}
	ctx := context.TODO()
	lk, err := c.lakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	if err := lk.RemovePool(ctx, name); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("pool deleted: %s\n", name)
	}
	return nil
}
