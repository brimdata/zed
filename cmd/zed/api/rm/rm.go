package rm

import (
	"errors"
	"flag"
	"fmt"

	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Rm = &charm.Spec{
	Name:  "rm",
	Usage: "rm [spacename]",
	Short: "removes a space",
	Long:  `The rm command takes a single argument and deletes the space`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{Command: parent.(*apicmd.Command)}, nil
	},
}

func init() {
	apicmd.Cmd.Add(Rm)
}

type Command struct {
	*apicmd.Command
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	if len(args) == 1 {
		c.Spacename = args[0]
	}
	id, err := c.SpaceID(ctx)
	if err != nil {
		return err
	}
	if err := c.Connection().SpaceDelete(ctx, id); err != nil {
		return err
	}
	name := c.Spacename
	if name == "" {
		name = string(id)
	}
	fmt.Printf("%s: space removed\n", c.Spacename)
	return nil
}
