package search

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/poolflags"
	devvector "github.com/brimdata/zed/cmd/zed/dev/vector"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zbuf"
)

var search = &charm.Spec{
	Name:  "search",
	Usage: "search [flags] filter_expr",
	Short: "run a VNG optimized search on a lake",
	New:   newCommand,
}

func init() {
	devvector.Cmd.Add(search)
}

type Command struct {
	*root.Command
	outputFlags outputflags.Flags
	poolFlags   poolflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.outputFlags.SetFlags(f)
	c.poolFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("usage: filter expression")
	}
	lk, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	root := lk.Root()
	if root == nil {
		return errors.New("remote lakes not supported")
	}
	head, err := c.poolFlags.HEAD()
	if err != nil {
		return err
	}
	text := args[0]
	rctx := runtime.NewContext(ctx, zed.NewContext())
	puller, err := compiler.VectorFilterCompile(rctx, text, data.NewSource(nil, root), head)
	if err != nil {
		return err
	}
	writer, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	if err := zbuf.CopyPuller(writer, puller); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
