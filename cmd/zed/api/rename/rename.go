package rename

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/api"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/segmentio/ksuid"
)

var Rename = &charm.Spec{
	Name:  "rename",
	Usage: "rename [old_name] new_name",
	Short: "renames a pool",
	Long:  `Renames a pool, given the current pool name and a desired new name.`,
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
	var id ksuid.KSUID
	var newname string
	switch len(args) {
	case 2:
		id, err = apicmd.LookupPoolID(ctx, c.Conn, args[0])
		if err != nil {
			return err
		}
		newname = args[1]
	case 1:
		if c.PoolName == "" {
			return errors.New("pool not specified")
		}
		id = c.PoolID
		newname = args[0]
	default:
		return errors.New("rename takes 1 or 2 arguments")
	}

	if err := c.Conn.PoolPut(ctx, id, api.PoolPutRequest{Name: newname}); err != nil {
		return err
	}
	fmt.Printf("pool renamed to %s\n", newname)
	return nil
}
