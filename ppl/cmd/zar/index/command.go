package index

import (
	"flag"

	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/ppl/cmd/zar/root"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [subcommand]",
	Short: "perform index related tasks on an archive",
	New:   New,
}

func init() {
	Index.Add(Create)
	Index.Add(Drop)
	Index.Add(Ls)
	root.Zar.Add(Index)
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{parent.(*root.Command)}, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return root.Zar.Exec(c.Command, []string{"help", "index"})
	}
	return charm.ErrNoRun
}
