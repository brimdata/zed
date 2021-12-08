package rename

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "rename",
	Usage: "rename old-name new-name",
	Short: "rename a data pool",
	Long: `
The rename command changes the name of the pool given by the -p option to the
new name provided.
`,
	New: New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.LakeFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 2 {
		return errors.New("two pool names must be provided")
	}
	oldName := args[0]
	newName := args[1]
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	poolID, err := lake.PoolID(ctx, oldName)
	if err != nil {
		return err
	}
	if err := lake.RenamePool(ctx, poolID, newName); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("pool %s renamed from %s to %s\n", poolID, oldName, newName)
	}
	return nil
}
