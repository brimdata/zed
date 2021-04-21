package root

import (
	"context"
	"flag"
	"os"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/signalctx"
)

var Zed = &charm.Spec{
	Name:  "zed",
	Usage: "zed <command> [options] [arguments...]",
	Short: "run zed commands",
	Long: `
zed is a command-line tool for creating, configuring, ingesting into,
querying, and orchestrating Zed data lakes.`,
	New: New,
}

type Command struct {
	charm.Command
	cli cli.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.cli.SetFlags(f)
	return c, nil
}

func (c *Command) Init(all ...cli.Initializer) (context.Context, func(), error) {
	if err := c.cli.Init(all...); err != nil {
		return nil, nil, err
	}
	ctx, cancel := signalctx.New(os.Interrupt)
	var cleanup = func() {
		cancel()
		c.cli.Cleanup()
	}
	return ctx, cleanup, nil
}

func (c *Command) Run(args []string) error {
	defer c.cli.Cleanup()
	if err := c.cli.Init(); err != nil {
		return err
	}
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
