package root

import (
	"flag"

	"github.com/mccanne/charm"
)

var MicroIndex = &charm.Spec{
	Name:  "microindex",
	Usage: "microindex <command> [options] [arguments...]",
	Short: "create and manipulate microindexes",
	Long: `
microindex is command-line utility for creating and manipulating microindexes.`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{}, nil
	},
}

func init() {
	MicroIndex.Add(charm.Help)
}

type Command struct{}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return MicroIndex.Exec(c, []string{"help"})
	}
	return charm.ErrNoRun
}
