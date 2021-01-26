package rename

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/mccanne/charm"
)

var Rename = &charm.Spec{
	Name:  "rename",
	Usage: "rename [old_name] [new_name]",
	Short: "renames a space",
	Long:  `Renames a space, given the current space name and a desired new name.`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{Command: parent.(*cmd.Command)}, nil
	},
}

func init() {
	cmd.CLI.Add(Rename)
}

type Command struct {
	*cmd.Command
}

func (c *Command) Run(args []string) error {
	if len(args) != 2 {
		return errors.New("expected <old_name> <new_name>")
	}
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	oldname := args[0]
	newname := args[1]
	id, err := cmd.GetSpaceID(c.Context(), c.Connection(), oldname)
	if err != nil {
		return err
	}
	if err := c.Connection().SpacePut(c.Context(), id, api.SpacePutRequest{Name: newname}); err != nil {
		return err
	}
	fmt.Printf("%s: space renamed to %s\n", oldname, newname)
	return nil
}
