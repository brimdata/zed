package index

import (
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [-R root] [options]",
	Short: "list and display lake index rules",
	New:   NewLs,
}

type LsCommand struct {
	lake        *zedlake.Command
	outputFlags outputflags.Flags
}

func NewLs(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LsCommand{lake: parent.(*Command).Command}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *LsCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	root, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	w, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	defer w.Close()
	return root.ScanIndex(ctx, w, root.ListIndexIDs())
}
