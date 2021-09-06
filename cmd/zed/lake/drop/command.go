package drop

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "drop",
	Usage: "drop pool",
	Short: "delete a data pool from a lake",
	Long: `
The drop command removes the named pool from the lake.

DANGER ZONE.
When deleting an entire pool, the drop command prompts for confirmation.
Once the pool is deleted, its data is gone so use this command carefully.
`,
	New: New,
}

func init() {
	zedapi.Cmd.Add(Cmd)
	zedlake.Cmd.Add(Cmd)
}

type Command struct {
	lake      zedlake.Command
	force     bool
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.BoolVar(&c.force, "f", false, "do not prompt for confirmation")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("a single pool name must be specified")
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolName := args[0]
	poolID, err := lake.PoolID(ctx, poolName)
	if err != nil {
		return nil
	}
	if err := c.confirm(poolName); err != nil {
		return err
	}
	if err := lake.RemovePool(ctx, poolID); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("pool deleted: %s\n", poolName)
	}
	return nil
}

func (c *Command) confirm(name string) error {
	if c.force {
		return nil
	}
	fmt.Printf("Are you sure you want to delete pool %q? There is no going back... [y|n]\n", name)
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return err
	}
	input = strings.ToLower(input)
	if input == "y" || input == "yes" {
		return nil
	}
	return errors.New("operation canceled")
}
