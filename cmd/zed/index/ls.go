package index

import (
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
)

var ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [-R root] [options]",
	Short: "list and display lake index rules",
	New:   newLs,
}

type lsCommand struct {
	*Command
	outputFlags outputflags.Flags
}

func newLs(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &lsCommand{Command: parent.(*Command)}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *lsCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	defer w.Close()
	r, err := api.ScanIndexRules(ctx, lake)
	if err != nil {
		return err
	}
	defer r.Close()
	return zio.Copy(w, r)
}
