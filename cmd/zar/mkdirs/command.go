package mkdirs

import (
	"errors"
	"flag"
	"regexp"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/reglob"
	"github.com/mccanne/charm"
)

var MkDirs = &charm.Spec{
	Name:  "mkdirs",
	Usage: "mkdirs [-p <glob>] dir",
	Short: "walk a directory tree and create zar directories",
	Long: `
"zar mkdirs" descends the directory given by the dir argument looking for
log files that match the glob-style pattern specified by -p.
If -p is not provided, "*.zng" is used as a default.  For each matched file,
"zar mkdirs" creates a zar directory if it does not already exist.  This conveniently
separates the problem of identifying the log files that should have zar
directories from the commands that operate on the content of the directories.
Each zar directory path name is comprised of its log file path concatenated
with the extension ".zar".
`,
	New: New,
}

func init() {
	root.Zar.Add(MkDirs)
}

type Command struct {
	*root.Command
	glob string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.glob, "p", "*.zng", "glob pattern for logs")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar mkdirs: directory to walk must be specified")
	}
	re, err := regexp.Compile(reglob.Reglob(c.glob))
	if err != nil {
		return err
	}
	return archive.MkDirs(args[0], re)
}
