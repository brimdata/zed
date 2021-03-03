package intake

import (
	"flag"
	"fmt"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cli/outputflags"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/mccanne/charm"
)

var Update = &charm.Spec{
	Name:  "update",
	Usage: "intake update <intake-name-or-id>",
	Short: "update an intake's configuration",
	Long: `
"intake update" can be used to change the configuration for an intake, including
changing the configured shaper, target, or name. If desired, the intake's shaper
and target space may be cleared by specifying an empty string for either.
`,
	New: NewUpdate,
}

type UpdateCommand struct {
	*Command
	flagSet     *flag.FlagSet
	outputFlags outputflags.Flags

	name        string
	shaper      string
	targetSpace string
}

func NewUpdate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &UpdateCommand{Command: parent.(*Command)}
	c.flagSet = f
	c.outputFlags.DefaultFormat = "table"
	c.outputFlags.SetFormatFlags(f)

	f.StringVar(&c.name, "name", "", "intake name")
	f.StringVar(&c.shaper, "shaper", "", "intake Z shaper code")
	f.StringVar(&c.targetSpace, "target", "", "intake target space (name or id)")
	return c, nil
}

func (c *UpdateCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("expected one argument of intake id or name")
	}
	intake, err := c.lookupIntake(args[0])
	if err != nil {
		return err
	}
	req := api.IntakePostRequest{
		Name:          intake.Name,
		Shaper:        intake.Shaper,
		TargetSpaceID: intake.TargetSpaceID,
	}
	err = flagVisit(c.flagSet, func(f *flag.Flag) error {
		switch f.Name {
		case "name":
			req.Name = c.name
		case "shaper":
			req.Shaper = c.shaper
		case "target":
			space, err := c.lookupSpace(c.targetSpace)
			if err != nil {
				return err
			}
			req.TargetSpaceID = space.ID
		}
		return nil
	})
	if err != nil {
		return err
	}
	intake, err = c.Connection().IntakeUpdate(c.Context(), intake.ID, req)
	if err != nil {
		return err
	}
	return cmd.WriteOutput(c.Context(), c.outputFlags, newIntakeReader([]api.Intake{intake}))
}

func flagVisit(fs *flag.FlagSet, fn func(*flag.Flag) error) error {
	var err error
	fs.Visit(func(f *flag.Flag) {
		if err == nil {
			err = fn(f)
		}
	})
	return err
}
