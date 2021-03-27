package idx

import (
	"flag"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/pkg/charm"
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

type Command struct {
	*cmd.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*cmd.Command)}, nil
}

func (c *Command) Run(args []string) error {
	return charm.ErrNoRun
}
