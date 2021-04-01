package compact

import (
	"context"
	"flag"
	"os"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Compact = &charm.Spec{
	Name:  "compact",
	Usage: "compact [-R root]",
	Short: "merge overlapping chunk files",
	Long: `
"zar compact" looks for chunk files whose time ranges overlap, and writes
new chunk files that combine their records.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Compact)
}

type Command struct {
	*zedlake.Command
	root  string
	purge bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZED_LAKE_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.purge, "purge", false, "remove chunk files (and associated files) whose data has been combined into other chunks")
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}
	ctx := context.TODO()
	if err := lake.Compact(ctx, lk, nil); err != nil {
		return err
	}
	if c.purge {
		return lake.Purge(ctx, lk, nil)
	}
	return nil
}
