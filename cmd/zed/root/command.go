package root

import (
	"flag"

	"github.com/brimdata/super/cli"
	"github.com/brimdata/super/cli/lakeflags"
	"github.com/brimdata/super/pkg/charm"
)

var Zed = &charm.Spec{
	Name:  "zed",
	Usage: "zed <command> [options] [arguments...]",
	Short: "run Zed commands",
	Long: `
zed is a command-line tool for creating, configuring, ingesting into,
querying, and orchestrating Zed data lakes.`,
	New: New,
}

type Command struct {
	charm.Command
	cli.Flags
	LakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}
	c.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	_, cancel, err := c.Init()
	if err != nil {
		return err
	}
	defer cancel()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
