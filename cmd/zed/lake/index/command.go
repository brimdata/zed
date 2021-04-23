package index

import (
	"flag"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [subcommand]",
	Short: "create and drop index rules, index data",
	New:   New,
}

func init() {
	Index.Add(Add)
	Index.Add(Create)
	Index.Add(Drop)
	Index.Add(Ls)
	zedlake.Cmd.Add(Index)
}

type Command struct {
	*zedlake.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{parent.(*zedlake.Command)}, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
