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
	Usage: "mkdirs [-d <dir>] [glob]",
	Short: "walk a directory tree and create zar directories",
	Long: `
"zar mkdirs" descends the directory given by the -d option looking for
log files that match the glob-style expression.  If the glob is not provided,
"*.zng" is used as a default.  For each matched file,
mkdirs creates a zar directory if it does not already exist.  This conveniently
separates the problem of identifying the log files that should have zar
directories from the commands that operate on the content of the directories.
The zar directory's path name is comprised of the file path concatenated
with the extension ".zar".
`,
	New: New,
}

func init() {
	root.Zar.Add(MkDirs)
}

type Command struct {
	*root.Command
	dir string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.dir, "d", ".", "directory to descend")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 1 {
		return errors.New("zar mkdirs: too many arguments")
	}
	pattern := "*.zng"
	if len(args) == 1 {
		pattern = args[0]
	}
	re, err := regexp.Compile(reglob.Reglob(pattern))
	if err != nil {
		return err
	}
	return archive.MkDirs(c.dir, re)
}
