package idx

import (
	"flag"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/mccanne/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [find|create]",
	Short: "query and create search indexes",
	Long:  "",
	New:   New,
}

func init() {
	cmd.CLI.Add(Index)
	Index.Add(Find)
	Index.Add(Create)
}

type IndexCmd struct {
	*cmd.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &IndexCmd{Command: parent.(*cmd.Command)}, nil
}

func (c *IndexCmd) Run(args []string) error {
	return charm.ErrNoRun
}
