package intake

import (
	"flag"

	"github.com/brimdata/zq/cli/outputflags"
	apicmd "github.com/brimdata/zq/cmd/zed/api"
	"github.com/brimdata/zq/pkg/charm"
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
	defer c.Cleanup()
	if err := c.Init(&c.outputFlags); err != nil {
		return err
	}
	conn := c.Connection()
	intakes, err := conn.IntakeList(c.Context())
	if err != nil {
		return err
	}
	if c.lflag {
		return apicmd.WriteOutput(c.Context(), c.outputFlags, newIntakeReader(intakes))
	}
	names := make([]string, 0, len(intakes))
	for _, n := range intakes {
		names = append(names, n.Name)
	}
	return apicmd.WriteOutput(c.Context(), c.outputFlags, apicmd.NewNameReader(names))
}
