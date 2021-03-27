package zst

import (
	"flag"

	"github.com/brimsec/zq/cli"
	"github.com/brimsec/zq/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "zst",
	Usage: "zst <command> [options] [arguments...]",
	Short: "create and manipulate zst columnar objects",
	Long: `
zst is a command-line tool for creating and manipulating zst columnar objects.`,
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
