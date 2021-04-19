package rename

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/api"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
)

var Rename = &charm.Spec{
	Name:  "rename",
	Usage: "rename [old_name] new_name",
	Short: "renames a space",
	Long:  `Renames a space, given the current space name and a desired new name.`,
	New: func(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
		return &Command{Command: parent.(*apicmd.Command)}, nil
	},
}

func init() {
	apicmd.Cmd.Add(Rename)
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
	var id api.SpaceID
	var newname string
	switch len(args) {
	case 2:
		newname = args[1]
		id, err = apicmd.GetSpaceID(ctx, c.Connection(), args[0])
		if err != nil {
			return err
		}
	case 1:
		newname = args[0]
		id, err = c.SpaceID(ctx)
		if err != nil {
			return err
		}
	default:
		return errors.New("rename takes 1 or 2 arguments")
	}

	if err := c.Connection().SpacePut(ctx, id, api.SpacePutRequest{Name: newname}); err != nil {
		return err
	}
	fmt.Printf("space renamed to %s\n", newname)
	return nil
}
