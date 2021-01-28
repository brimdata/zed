package rm

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/mccanne/charm"
)

var Rm = &charm.Spec{
	Name:  "rm",
	Usage: "rm [spacename]",
	Short: "removes a space",
	Long:  `The rm command takes a single argument and deletes the space`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{Command: parent.(*cmd.Command)}, nil
	},
}

func init() {
	cmd.CLI.Add(Rm)
}

type Command struct {
	*cmd.Command
}

func (c *Command) Run(args []string) error {
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	if len(args) == 1 {
		c.Spacename = args[0]
	}
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	id, err := c.SpaceID()
	if err != nil {
		return err
	}
	if err := c.Connection().SpaceDelete(c.Context(), id); err != nil {
		return err
	}
	name := c.Spacename
	if name == "" {
		name = string(id)
	}
	fmt.Printf("%s: space removed\n", c.Spacename)
	return nil
}
