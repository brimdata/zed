package create

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/units"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-orderby key[,key...][:asc|:desc]] -p name",
	Short: "create a new data pool",
	Long: `
The lake create command creates new pools.  One or more pool keys may be specified
as the sort keys (primary, secondary, etc) of the data stored in the pool.
The prefix ":asc" or ":desc" appearing after the comma-separated list of
keys indicates the sort order.  If no sort order is given, ascending is assumed.

The -p option is required and specifies the name for the pool.

The lake query command can efficiently perform
range scans with respect to the pool key using the
"range" parameter to the Zed "from" operator as the data is laid out
naturally for such scans.

By default, a branch called "main" is initialized in the newly created pool.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Create)
	zedapi.Cmd.Add(Create)
}

type Command struct {
	lake      zedlake.Command
	layout    string
	thresh    units.Bytes
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	c.thresh = data.DefaultThreshold
	f.Var(&c.thresh, "S", "target size of pool data objects, as '10MB' or '4GiB', etc.")
	f.StringVar(&c.layout, "orderby", "ts:desc", "comma-separated pool keys with optional :asc or :desc suffix to organize data in pool (cannot be changed)")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 0 {
		return errors.New("create command does not take arguments")
	}
	poolName, branchName := c.lakeFlags.Names()
	if poolName == "" {
		return errors.New("a pool or branch must be specified with -p")
	}
	if branchName != "" {
		return errors.New("branch cannot be specified with pool name; use branch command to create branches")
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	layout, err := order.ParseLayout(c.layout)
	if err != nil {
		return err
	}
	id, err := lake.CreatePool(ctx, poolName, layout, int64(c.thresh))
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("pool created: %s %s\n", poolName, id)
	}
	return nil
}
