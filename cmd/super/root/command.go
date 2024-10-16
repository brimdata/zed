package root

import (
	"flag"

	"github.com/brimdata/super/cli"
	"github.com/brimdata/super/pkg/charm"
)

var Super = &charm.Spec{
	Name:  "super",
	Usage: "super <command> [options] [arguments...]",
	Short: "XXX run Zed commands",
	Long: `
XXX zed is a command-line tool for creating, configuring, ingesting into,
querying, and orchestrating Zed data lakes.`,
	New: New,
}

type Command struct {
	charm.Command
	cli.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	_, cancel, err := c.Init()
	if err != nil {
		return err
	}
	defer cancel()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
