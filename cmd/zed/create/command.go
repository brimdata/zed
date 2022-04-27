package create

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/units"
)

var Cmd = &charm.Spec{
	Name:  "create",
	Usage: "create [-orderby key[,key...][:asc|:desc]] name",
	Short: "create a new data pool",
	Long: `
The lake create command creates new pools.  One or more pool keys may be specified
as the sort keys (primary, secondary, etc) of the data stored in the pool.
The prefix ":asc" or ":desc" appearing after the comma-separated list of
keys indicates the sort order.  If no sort order is given, ascending is assumed.

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
	layout     string
	thresh     units.Bytes
	seekStride units.Bytes
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command:    parent.(*root.Command),
		seekStride: units.Bytes(data.DefaultSeekStride),
	}
	c.thresh = data.DefaultThreshold
	f.Var(&c.thresh, "S", "target size of pool data objects, as '10MB' or '4GiB', etc.")
	f.StringVar(&c.layout, "orderby", "ts:desc", "comma-separated pool keys with optional :asc or :desc suffix to organize data in pool (cannot be changed)")
	f.Var(&c.seekStride, "seekstride", "size of seek-index unit for ZNG data, as '32KB', '1MB', etc.")
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
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	layout, err := order.ParseLayout(c.layout)
	if err != nil {
		return err
	}
	poolName := args[0]
	id, err := lake.CreatePool(ctx, poolName, layout, int(c.seekStride), int64(c.thresh))
	if err != nil {
		return err
	}
	if !c.Quiet {
		fmt.Printf("pool created: %s %s\n", poolName, id)
	}
	return nil
}
