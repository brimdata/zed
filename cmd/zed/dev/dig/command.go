package dig

import (
	"flag"

	"github.com/brimdata/zed/cmd/zed/dev"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "dig",
	Usage: "dig sub-command [arguments...]",
	Short: "extract useful information from Zed streams or files",
	Long: `
The dig command provide various debug and test functions regarding the Zed family
of formats. When run with no arguments or -h, it lists help for the dig sub-commands.`,
	New: New,
}

func init() {
	dev.Cmd.Add(Cmd)
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*root.Command)}, nil
}

func (c *Command) Run(args []string) error {
	return charm.NeedHelp
}
