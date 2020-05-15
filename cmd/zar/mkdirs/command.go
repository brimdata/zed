package mkdirs

import (
	"flag"
	"os"
	"regexp"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/reglob"
	"github.com/mccanne/charm"
)

var MkDirs = &charm.Spec{
	Name:  "mkdirs",
	Usage: "mkdirs [-R archive] [-p <glob>]",
	Short: "walk a directory tree and create zar directories",
	Long: `
"zar mkdirs" walks an archive's directories looking for
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
	root string
	glob string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.StringVar(&c.glob, "p", "*.zng", "glob pattern for logs")
	return c, nil
}

func (c *Command) Run(args []string) error {
	re, err := regexp.Compile(reglob.Reglob(c.glob))
	if err != nil {
		return err
	}

	ark, err := archive.OpenArchive(c.root)
	if err != nil {
		return err
	}
	return archive.MkDirs(ark, re)
}
