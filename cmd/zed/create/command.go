package create

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/poolflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/units"
)

var Cmd = &charm.Spec{
	Name:  "create",
	Usage: "create [-orderby key[:asc|:desc]] name",
	Short: "create a new data pool",
	Long: `
The lake create command creates new pools.  A pool key may be specified
as the sort key of the data stored in the pool. The prefix ":asc" or ":desc"
appearing after the specified key indicates the sort order.  If no sort
order is given, ascending is assumed.

The single argument specifies the name for the pool.

The lake query command can efficiently perform
range scans with respect to the pool key using the
"range" parameter to the Zed "from" operator as the data is laid out
naturally for such scans.

By default, a branch called "main" is initialized in the newly created pool.
`,
	HiddenFlags: "seekstride",
	New:         New,
}

type Command struct {
	*root.Command
	sortKey    string
	thresh     units.Bytes
	seekStride units.Bytes
	use        bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command:    parent.(*root.Command),
		seekStride: units.Bytes(data.DefaultSeekStride),
	}
	f.Var(&c.seekStride, "seekstride", "size of seek-index unit for ZNG data, as '32KB', '1MB', etc.")
	c.thresh = data.DefaultThreshold
	f.Var(&c.thresh, "S", "target size of pool data objects, as '10MB' or '4GiB', etc.")
	f.BoolVar(&c.use, "use", false, "set created pool as the current pool")
	f.StringVar(&c.sortKey, "orderby", "ts:desc", "pool key with optional :asc or :desc suffix to organize data in pool (cannot be changed)")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("create requires one argument")
	}
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	sortKey, err := order.ParseSortKey(c.sortKey)
	if err != nil {
		return err
	}
	poolName := args[0]
	id, err := lake.CreatePool(ctx, poolName, sortKey, int(c.seekStride), int64(c.thresh))
	if err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		fmt.Printf("pool created: %s %s\n", poolName, id)
	}
	if c.use {
		if err := poolflags.WriteHead(poolName, "main"); err != nil {
			return err
		}
		if !c.LakeFlags.Quiet {
			fmt.Printf("Switched to branch \"main\" on pool %q\n", poolName)
		}
	}
	return nil
}
