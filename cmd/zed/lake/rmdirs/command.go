package rmdirs

import (
	"context"
	"flag"
	"os"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var RmDirs = &charm.Spec{
	Name:  "rmdirs",
	Usage: "rmdirs [-R root]",
	Short: "walk a directory tree and remove zar directories",
	Long: `
"zar rmdirs" descends the provided directory looking for
zar directories and removes them along with their contents.  WARNING:
this is no prompting for the files and directories that will be removed
so use carefully.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(RmDirs)
}

type Command struct {
	*zedlake.Command
	root string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}
	return lake.RmDirs(context.TODO(), lk)
}
