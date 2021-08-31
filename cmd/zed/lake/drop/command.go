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
	"github.com/segmentio/ksuid"
)

var Cmd = &charm.Spec{
	Name:  "drop",
	Usage: "drop -p pool[/branch]",
	Short: "delete a data pool from a lake",
	Long: `
"zed lake drop" removes the named pool from the lake.
The -p flag must be given.  If a branch is specified,
the branch is deleted and the pool remains.

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
	poolName, branchName := c.lakeFlags.Names()
	if poolName == "" {
		return errors.New("name of pool must be supplied with -p option")
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolID, branchID, err := lake.IDs(ctx, poolName, branchName)
	if err != nil {
		return nil
	}
	if branchID != ksuid.Nil {
		if err := lake.RemoveBranch(ctx, poolID, branchID); err != nil {
			return err
		}
		if !c.lakeFlags.Quiet {
			fmt.Printf("branch deleted: %s\n", branchName)
		}
		return nil
	}
	if err := c.confirm(); err != nil {
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

func (c *Command) confirm() error {
	if c.force {
		return nil
	}
	name := c.lakeFlags.PoolName
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
