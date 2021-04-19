package intake

import (
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "intake ls [-l]",
	Short: "list intakes",
	New:   NewLs,
}

type LsCommand struct {
	*Command
	lflag       bool
	outputFlags outputflags.Flags
}

func NewLs(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LsCommand{Command: parent.(*Command)}
	f.BoolVar(&c.lflag, "l", false, "output full information for each intake")
	c.outputFlags.DefaultFormat = "text"
	c.outputFlags.SetFormatFlags(f)
	return c, nil
}

func (c *LsCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	conn := c.Connection()
	intakes, err := conn.IntakeList(ctx)
	if err != nil {
		return err
	}
	if c.lflag {
		return apicmd.WriteOutput(ctx, c.outputFlags, newIntakeReader(intakes))
	}
	names := make([]string, 0, len(intakes))
	for _, n := range intakes {
		names = append(names, n.Name)
	}
	return apicmd.WriteOutput(ctx, c.outputFlags, apicmd.NewNameReader(names))
}
