package rename

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Rename = &charm.Spec{
	Name:  "rename",
	Usage: "rename -p old-name new-name",
	Short: "rename a data pool",
	Long: `
The rename command changes the name of the pool given by the -p option to the
new name provided.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Rename)
	zedapi.Cmd.Add(Rename)
}

type Command struct {
	lake      zedlake.Command
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	oldName := c.lakeFlags.PoolName
	if oldName == "" {
		return errors.New("rename pool: -p required")
	}
	if len(args) > 1 {
		return errors.New("rename pool: too many arguments")
	}
	if len(args) != 1 {
		return errors.New("rename pool: new name of pool is required")
	}
	newName := args[0]
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolID, _, err := lake.IDs(ctx, oldName, "main")
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
