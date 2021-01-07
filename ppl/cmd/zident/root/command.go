package root

import (
	"flag"

	"github.com/mccanne/charm"
)

var Zident = &charm.Spec{
	Name:  "zident",
	Usage: "zident [global options] command [options] [arguments...]",
	Short: "tenant and user utility",
	Long: `
zident is a utility to interact with the systems used for tenant and user
authentication and authorization in a zqd service instance.
`,
	New: New,
}

type Command struct {
	charm.Command
}

func init() {
	Zident.Add(charm.Help)
}

func New(_ charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{}, nil
}

func (c *Command) Run(args []string) error {
	return charm.ErrNoRun
}
