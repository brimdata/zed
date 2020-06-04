package rmdirs

import (
	"flag"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var RmDirs = &charm.Spec{
	Name:  "rmdirs",
	Usage: "rmdirs [-R archive]",
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
	root.Zar.Add(RmDirs)
}

type Command struct {
	*root.Command
	root string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}
	return archive.RmDirs(ark)
}
