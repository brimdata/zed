package root

import (
	"context"
	"flag"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/pkg/charm"
)

var Zed = &charm.Spec{
	Name:  "zed",
	Usage: "zed <command> [options] [arguments...]",
	Short: "run Zed commands",
	Long: `
zed is a command-line tool for creating, configuring, ingesting into,
querying, and orchestrating Zed data lakes.`,
	New: New,
}

type Command struct {
	charm.Command
	LakeFlags lakeflags.Flags
	cli       cli.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.cli.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Init(all ...cli.Initializer) (context.Context, func(), error) {
	return c.cli.Init(all...)
}

func (c *Command) Run(args []string) error {
	_, cancel, err := c.cli.Init()
	if err != nil {
		return err
	}
	defer cancel()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
