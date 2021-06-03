package ls

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [options]",
	Short: "list objects in data pool",
	Long: `
"zed lake ls" shows a listing of a data pool's segments as tags.
The the pool flag "-p" is not given, then the lake's pools are listed.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Ls)
	zedapi.Cmd.Add(Ls)
}

type Command struct {
	lake        *zedlake.Command
	partition   bool
	at          string
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.StringVar(&c.at, "at", "", "commit tag or journal ID for time travel")
	f.BoolVar(&c.partition, "partition", false, "display partitions as determined by scan logic")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 0 {
		return errors.New("zed lake ls: too many arguments")
	}
	ctx, cleanup, err := c.lake.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	local := storage.NewLocalEngine()
	if c.lake.Flags.PoolName() == "" {
		lk, err := c.lake.Flags.Open(ctx)
		if err != nil {
			return err
		}
		zw, err := c.outputFlags.Open(ctx, local)
		if err != nil {
			return err
		}
		defer zw.Close()
		return lk.ScanPools(ctx, zw)
	}
	pool, err := c.lake.Flags.OpenPool(ctx)
	if err != nil {
		return err
	}
	zw, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	defer zw.Close()
	return pool.ScanSegments(ctx, zw, c.at, c.partition, nil)
}
