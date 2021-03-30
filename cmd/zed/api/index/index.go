package idx

import (
	"flag"

	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index [find|create]",
	Short: "query and create search indexes",
	Long:  "",
	New:   New,
}

func init() {
	apicmd.Cmd.Add(Index)
	Index.Add(Find)
	Index.Add(Create)
}

type Command struct {
	*apicmd.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*apicmd.Command)}, nil
}

func (c *Command) Run(args []string) error {
	return charm.ErrNoRun
}
