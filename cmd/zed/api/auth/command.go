package auth

import (
	"flag"

	"github.com/brimsec/zq/cmd/zed/api"
	"github.com/brimsec/zq/pkg/charm"
)

var Auth = &charm.Spec{
	Name:  "auth",
	Usage: "auth [subcommand]",
	Short: "authentication and authorization related commands",
	Long:  ``,
	New:   New,
	// Marking auth & subcommands hidden until support plumbed through all
	// operations, see zq#1887 .
	Hidden: true,
}

func init() {
	Auth.Add(Login)
	Auth.Add(Logout)
	Auth.Add(Method)
	Auth.Add(Store)
	Auth.Add(Verify)
	api.Cmd.Add(Auth)
}

type Command struct {
	*api.Command
	AuthToken string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*api.Command)}, nil
}

func (c *Command) Run(args []string) error {
	return charm.ErrNoRun
}
