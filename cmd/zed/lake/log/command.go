package log

import (
	"context"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zbuf"
)

var Log = &charm.Spec{
	Name:  "log",
	Usage: "log [-R root] [options] [pattern]",
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
	*zedlake.Command
	lk          *lake.Root
	lakeFlags   zedlake.Flags
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	c.outputFlags.DefaultFormat = "zson"
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.outputFlags); err != nil {
		return err
	}
	ctx := context.TODO()
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	reader, err := pool.Log().OpenAsZNG(ctx, 0, 0)
	if err != nil {
		return err
	}
	writer, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	if err := zbuf.Copy(writer, reader); err != nil {
		return err
	}
	return writer.Close()
}
