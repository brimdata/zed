package root

import (
	"flag"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/pkg/charm"
)

var CLI = &charm.Spec{
	Name:  "pgctl",
	Usage: "pgctl command [options] [arguments...]",
	Short: "administrative commands for a zqd postgres instance",
	New:   New,
}

type Command struct {
	charm.Command
	cli cli.Flags
}

func init() {
	CLI.Add(charm.Help)
}

func New(_ charm.Command, _ *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	return c, nil
}

func (c *Command) Init(all ...cli.Initializer) error {
	return c.cli.Init(all...)
}

func (c *Command) Run(args []string) error {
	return CLI.Exec(c, []string{"help"})
}
