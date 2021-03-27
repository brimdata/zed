package rename

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/cmd/zapi/cmd"
	"github.com/brimsec/zq/pkg/charm"
)

var Rename = &charm.Spec{
	Name:  "rename",
	Usage: "rename [old_name] new_name",
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
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}

	var err error
	var id api.SpaceID
	var newname string
	switch len(args) {
	case 2:
		newname = args[1]
		id, err = cmd.GetSpaceID(c.Context(), c.Connection(), args[0])
		if err != nil {
			return err
		}
	case 1:
		newname = args[0]
		id, err = c.SpaceID()
		if err != nil {
			return err
		}
	default:
		return errors.New("rename takes 1 or 2 arguments")
	}

	if err := c.Connection().SpacePut(c.Context(), id, api.SpacePutRequest{Name: newname}); err != nil {
		return err
	}
	fmt.Printf("space renamed to %s\n", newname)
	return nil
}
