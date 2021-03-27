package root

import (
	"flag"

	"github.com/brimsec/zq/cli"
	"github.com/brimsec/zq/pkg/charm"
)

var MicroIndex = &charm.Spec{
	Name:  "microindex",
	Usage: "microindex <command> [options] [arguments...]",
	Short: "create and manipulate microindexes",
	Long: `
microindex is command-line utility for creating and manipulating microindexes.`,
	New: New,
}

func init() {
	MicroIndex.Add(charm.Help)
}

type Command struct {
	charm.Command
	cli cli.Flags
}

func (c *Command) Cleanup() {
	c.cli.Cleanup()
}

func (c *Command) Init(all ...cli.Initializer) error {
	return c.cli.Init(all...)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.cli.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.cli.Cleanup()
	err := c.cli.Init()
	if len(args) == 0 {
		return MicroIndex.Exec(c, []string{"help"})
	}
	if err != nil {
		return err
	}
	return charm.ErrNoRun
}
