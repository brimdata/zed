package manage

import (
	"flag"

	"github.com/brimdata/zed/cli/commitflags"
	"github.com/brimdata/zed/cli/logflags"
	"github.com/brimdata/zed/cmd/zed/manage"
	"github.com/brimdata/zed/cmd/zed/manage/lakemanage"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "monitor",
	Usage: "monitor",
	Short: "monitor pools in a lake",
	New:   New,
}

func init() {
	manage.Cmd.Add(Cmd)
}

type Command struct {
	*manage.Command
	commitFlags commitflags.Flags
	manageFlags manage.Flags
	logFlags    logflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*manage.Command)}
	c.commitFlags.SetFlags(f)
	c.manageFlags.SetFlags(f)
	c.logFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	conn, err := c.LakeFlags.Connection()
	if err != nil {
		return err
	}
	logger, err := c.logFlags.Open()
	if err != nil {
		return err
	}
	defer logger.Sync()
	return lakemanage.Monitor(ctx, conn, c.manageFlags.Config, logger)
}
