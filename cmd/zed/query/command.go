package query

import (
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/cli/queryflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
)

var Cmd = &charm.Spec{
	Name:  "query",
	Usage: "query [options] [zed-query]",
	Short: "run a Zed query on a Zed data lake",
	Long: `
"zed query" runs a Zed query on a Zed data lake.
`,
	New: New,
}

type Command struct {
	*root.Command
	queryFlags  queryflags.Flags
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	c.queryFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 1 || len(args) == 0 && len(c.queryFlags.Includes) == 0 {
		return charm.NeedHelp
	}
	var src string
	if len(args) == 1 {
		src = args[0]
	}
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	head, _ := c.HEAD()
	query, err := lake.QueryWithControl(ctx, head, src, c.queryFlags.Includes...)
	if err != nil {
		w.Close()
		return err
	}
	defer query.Close()
	err = zio.Copy(w, zbuf.NoControl(query))
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		c.queryFlags.PrintStats(query.Progress())
	}
	return err
}
