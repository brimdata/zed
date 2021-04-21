package zst

import (
	"flag"

	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "zst",
	Usage: "zst <command> [options] [arguments...]",
	Short: "create and manipulate zst columnar objects",
	Long: `
zst is a command-line tool for creating and manipulating zst columnar objects.`,
	New: New,
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*root.Command)}, nil
}

func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
