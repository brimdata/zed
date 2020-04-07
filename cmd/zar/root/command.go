package root

import (
	"flag"

	"github.com/mccanne/charm"
)

var Zar = &charm.Spec{
	Name:  "zar",
	Usage: "zar [global options] command [options] [arguments...]",
	Short: "create and search zng archives",
	Long: `
zar creates and searches index files for zng files.
`,
	New: New,
}

type Command struct {
	charm.Command
}

func init() {
	Zar.Add(charm.Help)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{}, nil
}

func (c *Command) Run(args []string) error {
	return Zar.Exec(c, []string{"help"})
}
