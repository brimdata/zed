package root

import (
	"flag"

	"github.com/mccanne/charm"
)

var Sst = &charm.Spec{
	Name:  "sst",
	Usage: "sst <command> [options] [arguments...]",
	Short: "use sst to test/debug boom sst files",
	Long: `
sst is command-line utility useful for debugging the sst packaging and
interrogating sst files that are corrected by a client of sst.`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{}, nil
	},
}

func init() {
	Sst.Add(charm.Help)
}

type Command struct{}

func (c *Command) Run(args []string) error {
	return charm.ErrNoRun
}
