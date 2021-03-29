package intake

import (
	"flag"
	"fmt"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/cli/outputflags"
	apicmd "github.com/brimdata/zq/cmd/zed/api"
	"github.com/brimdata/zq/pkg/charm"
)

var NewSpec = &charm.Spec{
	Name:  "new",
	Usage: "intake new <name>",
	Short: "create a new intake",
	New:   NewNew,
}

type NewCommand struct {
	*Command
	outputFlags outputflags.Flags

	name        string
	shaper      string
	targetSpace string
}

func NewNew(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &NewCommand{Command: parent.(*Command)}
	c.outputFlags.DefaultFormat = "table"
	c.outputFlags.SetFormatFlags(f)

	f.StringVar(&c.shaper, "shaper", "", "intake Z shaper code")
	f.StringVar(&c.targetSpace, "target", "", "intake target space name or id")
	return c, nil
}

func (c *NewCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	conn := c.Connection()
	var req api.IntakePostRequest

	if len(args) != 1 {
		return fmt.Errorf("expected one argument: name of intake")
	}
	req.Name = args[0]
	req.Shaper = c.shaper
	if c.targetSpace != "" {
		targetSpace, err := c.lookupSpace(c.targetSpace)
		if err != nil {
			return err
		}
		req.TargetSpaceID = targetSpace.ID
	}
	intake, err := conn.IntakeCreate(c.Context(), req)
	if err != nil {
		return err
	}
	return apicmd.WriteOutput(c.Context(), c.outputFlags, newIntakeReader([]api.Intake{intake}))
}
