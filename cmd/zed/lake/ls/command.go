package ls

import (
	"context"
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
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
}

type Command struct {
	*zedlake.Command
	partition   bool
	lakeFlags   zedlake.Flags
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.BoolVar(&c.partition, "partition", false, "display partitions as determined by scan logic")
	c.outputFlags.DefaultFormat = "zson"
	c.outputFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.outputFlags); err != nil {
		return err
	}
	if len(args) > 0 {
		return errors.New("zed lake ls: too many arguments")
	}
	ctx := context.TODO()
	w, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	var r zbuf.Reader
	if c.lakeFlags.PoolName == "" {
		lk, err := c.lakeFlags.Open(ctx)
		if err != nil {
			return err
		}
		r = lk.List()
	} else {
		pool, err := c.lakeFlags.OpenPool(ctx)
		if err != nil {
			return err
		}
		head, err := pool.Log().Head(ctx)
		if err != nil {
			return err
		}
		if c.partition {
			r = lake.NewPartionReader(ctx, head, nano.MaxSpan)
		} else {
			r = lake.NewSegmentReader(ctx, head, nano.MaxSpan)
		}
	}
	err = zbuf.Copy(w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}
