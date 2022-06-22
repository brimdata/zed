package vector

import (
	"flag"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "vector",
	Usage: "vector [subcommand]",
	Short: "create and delete vectorized versions of lake data",
	Long: `
The vector subcommands control the creation, management, and deletion
of vectorized data in a Zed lake.
`, New: New,
}

func init() {
	Cmd.Add(add)
	Cmd.Add(del)
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*root.Command)}, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
