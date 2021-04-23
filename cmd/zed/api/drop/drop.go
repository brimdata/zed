package rm

import (
	"errors"
	"flag"
	"fmt"

	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Drop = &charm.Spec{
	Name:  "drop",
	Usage: "drop -p name",
	Short: "delete a data pool from a lake",
	Long: `
"zed lake drop" removes the named pool from the lake.
The -p flag must be given.

DANGER ZONE.
There is no prompting or second chances here so use carefully.
Once the pool is deleted, its data is gone.
`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{Command: parent.(*apicmd.Command)}, nil
	},
}

func init() {
	apicmd.Cmd.Add(Drop)
}

type Command struct {
	*apicmd.Command
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if c.PoolName == "" {
		return errors.New("pool must be specified (-p)")
	}
	if err := c.Conn.PoolDelete(ctx, c.PoolID); err != nil {
		return err
	}
	fmt.Printf("%s: pool dropped\n", c.PoolName)
	return nil
}
