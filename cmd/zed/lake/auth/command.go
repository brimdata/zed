package auth

import (
	"flag"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

const (
	credsFileName = "credentials.json"
)

var Auth = &charm.Spec{
	Name:  "auth",
	Usage: "auth [subcommand]",
	Short: "authentication and authorization commands",
	Long:  ``,
	New:   New,
}

func init() {
	Auth.Add(Login)
	Auth.Add(Logout)
	Auth.Add(Method)
	Auth.Add(Store)
	Auth.Add(Verify)
	zedapi.Cmd.Add(Auth)
}

type Command struct {
	lake      *zedapi.Command
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedapi.Command)}
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	return charm.ErrNoRun
}
