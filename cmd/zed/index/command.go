package index

import (
	"flag"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "index",
	Usage: "index <command> [options] [arguments...]",
	Short: "create and search zed indexes from zed streams or files",
	Long: `
"zed index" is command-line utility for creating and manipulating zed indexes.
`,
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
