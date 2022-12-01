package update

import (
	"flag"

	"github.com/brimdata/zed/cli/logflags"
	"github.com/brimdata/zed/cmd/zed/manage"
	"github.com/brimdata/zed/cmd/zed/manage/lakemanage"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "update",
	Usage: "update",
	Short: "compact and index pools and then exit",
	New:   New,
}

func init() {
	manage.Cmd.Add(Cmd)
}

type Command struct {
	*manage.Command
	logFlags    logflags.Flags
	manageFlags manage.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*manage.Command)}
	c.logFlags.SetFlags(f)
	c.manageFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	lk, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	logger, err := c.logFlags.Open()
	if err != nil {
		return err
	}
	defer logger.Sync()
	return lakemanage.Update(ctx, lk, c.manageFlags.Config, logger)
}
