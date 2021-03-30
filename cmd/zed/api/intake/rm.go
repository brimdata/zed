package intake

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed/pkg/charm"
)

var Rm = &charm.Spec{
	Name:  "rm",
	Usage: "intake rm <intake-name-or-id>",
	Short: "delete an intake",
	New:   NewRm,
}

type RmCommand struct {
	*Command
}

func NewRm(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &RmCommand{Command: parent.(*Command)}
	return c, nil
}

func (c *RmCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("expected one argument")
	}
	intake, err := c.lookupIntake(args[0])
	if err != nil {
		return err
	}
	return c.Connection().IntakeDelete(c.Context(), intake.ID)
}
