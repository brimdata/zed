package create

import (
	"errors"
	"flag"
	"fmt"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/units"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-orderby key[,key...][:asc|:desc]] -p name",
	Short: "create a new data pool",
	Long: `
"zed create" ...
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Create)
}

type Command struct {
	lake   *zedlake.Command
	layout string
	thresh units.Bytes
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	c.thresh = segment.DefaultThreshold
	f.Var(&c.thresh, "S", "target size of pool data objects, as '10MB' or '4GiB', etc.")
	f.StringVar(&c.layout, "orderby", "ts:desc", "comma-separated pool keys with optional :asc or :desc suffix to organize data in pool (cannot be changed)")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	name := c.lake.Flags.PoolName
	if len(args) != 0 && name != "" {
		return errors.New("zed lake create pool: does not take arguments")
	}
	if name == "" {
		return errors.New("zed lake create pool: -p required")
	}
	layout, err := order.ParseLayout(c.layout)
	if err != nil {
		return err
	}
	if _, err := c.lake.Flags.CreatePool(ctx, layout, int64(c.thresh)); err != nil {
		return err
	}
	if !c.lake.Flags.Quiet {
		fmt.Printf("pool created: %s\n", c.lake.Flags.PoolName)
	}
	return nil
}
