package root

import (
	"flag"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/pkg/charm"
)

var Zed = &charm.Spec{
	Name:  "zed",
	Usage: "zed <command> [options] [arguments...]",
	Short: "run zed commands",
	Long: `
zed is a command-line tool for creating, configuring, ingesting into,
querying, and orchestrating zed data lakes.`,
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

func (c *Command) Cleanup() {
	c.cli.Cleanup()
}

func (c *Command) Init(all ...cli.Initializer) error {
	return c.cli.Init(all...)
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
