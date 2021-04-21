package log

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Log = &charm.Spec{
	Name:  "log",
	Usage: "log [options] [pattern]",
	Short: "show a data pool's commit log",
	Long: `
"zed lake log" outputs a data pool's commit log in the format desired.
By default, output is in the ZNG format so that it can easily be piped
to zq or other tooling for analysis.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Log)
}

type Command struct {
	lake        *zedlake.Command
	lk          *lake.Root
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 0 {
		return errors.New("zed lake load: no arguments allowed")
	}
	pool, err := c.lake.Flags.OpenPool(ctx)
	if err != nil {
		return err
	}
	r, err := pool.Log().OpenAsZNG(ctx, 0, 0)
	if err != nil {
		return err
	}
	return zedlake.CopyToOutput(ctx, c.outputFlags, r)
}
