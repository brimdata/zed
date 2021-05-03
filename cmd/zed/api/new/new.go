package new

import (
	"errors"
	"flag"
	"fmt"

	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

var NewSpec = &charm.Spec{
	Name:  "new",
	Usage: "new [poolname]",
	Short: "create a new pool",
	Long: `The new command takes a single argument and creates a new, empty pool
named as specified.`,
	New: New,
}

func init() {
	apicmd.Cmd.Add(NewSpec)
}

type Command struct {
	*apicmd.Command
	createFlags apicmd.PoolCreateFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*apicmd.Command)}
	c.createFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	if len(args) != 1 && c.PoolName == "" {
		return errors.New("must specify a pool name")
	}
	name := args[0]
	sp, err := c.createFlags.Create(ctx, c.Conn, name)
	if err != nil {
		return fmt.Errorf("couldn't create new pool %s: %w", name, err)
	}
	fmt.Printf("%s: pool created\n", sp.Name)
	return nil
}
