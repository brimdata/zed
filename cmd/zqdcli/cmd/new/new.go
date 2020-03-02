package new

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zqdcli/cmd"
	"github.com/mccanne/charm"
)

var New = &charm.Spec{
	Name:  "new",
	Usage: "new [spacename]",
	Short: "create a new space",
	Long: `The new command takes a single argument and creates a new, empty space
named as specified.`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{Command: parent.(*cmd.Command)}, nil
	},
}

func init() {
	cmd.Cli.Add(New)
}

type Command struct {
	*cmd.Command
}

func (c *Command) Run(args []string) error {
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	if len(args) != 1 {
		return errors.New("must specify a space name")
	}
	api, err := c.API()
	if err != nil {
		return err
	}
	name := args[0]
	_, err = api.SpacePost(name)
	if err != nil {
		return fmt.Errorf("couldn't create new space %s: %v", name, err)
	}
	fmt.Printf("%s: space created\n", name)
	return nil
}
