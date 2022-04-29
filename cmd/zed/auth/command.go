package auth

import (
	"flag"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "auth",
	Usage: "auth [subcommand]",
	Short: "authentication and authorization commands",
	Long:  ``,
	New:   New,
}

func init() {
	Cmd.Add(Login)
	Cmd.Add(Logout)
	Cmd.Add(Method)
	Cmd.Add(Store)
	Cmd.Add(Verify)
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*root.Command)}, nil
}

func (c *Command) Run(args []string) error {
	return charm.ErrNoRun
}
