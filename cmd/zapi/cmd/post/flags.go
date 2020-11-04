package post

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/cmd/zapi/cmd"
)

type spaceFlags struct {
	cmd.SpaceCreateFlags
	force bool
	cmd   *cmd.Command
}

func (f *spaceFlags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.force, "f", false, "create space if specified space does not exist")
	f.SpaceCreateFlags.SetFlags(fs)
}

func (f *spaceFlags) Init() error {
	c := f.cmd
	if err := c.Init(&f.SpaceCreateFlags); err != nil {
		return err
	}
	if !f.force {
		return nil
	} else if c.Spacename == "" {
		return errors.New("if -f flag is enabled, a space name must specified")
	}

	sp, err := f.SpaceCreateFlags.Create(c.Context(), c.Connection(), c.Spacename)
	if err != nil && err != client.ErrSpaceExists {
		return err
	}
	if sp != nil {
		c.SetSpaceID(sp.ID)
	} else {
		// space already exists, fetch space ID
		_, err := c.SpaceID()
		return err
	}
	return nil
}
