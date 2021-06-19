package drop

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	zedapi "github.com/brimdata/zed/cmd/zed/api"
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
	zedapi.Cmd.Add(Cmd)
	zedlake.Cmd.Add(Cmd)
}

type Command struct {
	lake  *zedlake.Command
	force bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.BoolVar(&c.force, "f", false, "do not confirm pool deletion")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	name := c.lake.Flags.PoolName()
	if name == "" {
		return errors.New("name of pool must be supplied with -p option")
	}
	lk, err := c.lake.Flags.Open(ctx)
	if err != nil {
		return err
	}
	pool, err := lk.LookupPoolByName(ctx, name)
	if err != nil {
		return nil
	}
	if pool == nil {
		return fmt.Errorf("%s: no such pool", name)
	}
	if err := c.confirm(); err != nil {
		return err
	}
	if err := lk.RemovePool(ctx, pool.ID); err != nil {
		return err
	}
	if !c.lake.Flags.Quiet() {
		fmt.Printf("pool deleted: %s\n", name)
	}
	return nil
}

func (c *Command) confirm() error {
	if c.force {
		return nil
	}
	name := c.lake.Flags.PoolName()
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
