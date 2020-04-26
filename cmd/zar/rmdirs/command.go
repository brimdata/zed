package rmdirs

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var RmDirs = &charm.Spec{
	Name:  "rmdirs",
	Usage: "rmdirs [-d <dir>]",
	Short: "walk a directory tree and remove zar directories",
	Long: `
"zar rmdirs" descends the directory given by the -d option looking for
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
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar rmdirs: must specified top-level directory to walk and delete")
	}
	return archive.RmDirs(args[0])
}
