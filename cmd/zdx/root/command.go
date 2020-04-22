package root

import (
	"flag"

	"github.com/mccanne/charm"
)

var Zdx = &charm.Spec{
	Name:  "zdx",
	Usage: "zdx <command> [options] [arguments...]",
	Short: "use zdx to test/debug boom sst files",
	Long: `
zdx is command-line utility useful for debugging the zdx packaging and
interrogating zdx files that are corrected by a client of zdx.`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{}, nil
	},
}

func init() {
	Zdx.Add(charm.Help)
}

type Command struct{}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return Zdx.Exec(c, []string{"help"})
	}
	return charm.ErrNoRun
}
