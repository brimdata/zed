package ls

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/mccanne/charm"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [-R root] [options] [pattern]",
	Short: "list the zar directories in an archive",
	Long: `
"zar ls" walks an archive's directories and prints out
the path of each zar directory contained with those top-level directories.
`,
	New: New,
}

func init() {
	root.Zar.Add(Ls)
}

type Command struct {
	*root.Command
	root          string
	lflag         bool
	relativePaths bool
	showRanges    bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.lflag, "l", false, "long form")
	f.BoolVar(&c.relativePaths, "relative", false, "display paths relative to root")
	f.BoolVar(&c.showRanges, "ranges", false, "display time ranges instead of paths")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 1 {
		return errors.New("zar ls: too many arguments")
	}

	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}

	var pattern string
	if len(args) == 1 {
		pattern = args[0]
	}
	return archive.Walk(context.TODO(), ark, func(chunk archive.Chunk) error {
		c.printDir(ark, chunk, pattern)
		return nil
	})
}

func fileExists(path iosrc.URI) bool {
	info, err := iosrc.Stat(context.TODO(), path)
	if err != nil {
		return false
	}
	if fsinfo, ok := info.(os.FileInfo); ok {
		return !fsinfo.IsDir()
	}
	return true
}

func (c *Command) printDir(ark *archive.Archive, chunk archive.Chunk, pattern string) {
	dir := chunk.ZarDir(ark)
	if pattern != "" {
		path := dir.AppendPath(pattern)
		if fileExists(path) {
			fmt.Println(c.printable(ark, chunk, dir, pattern))
		}
		return
	}
	if !c.lflag {
		fmt.Println(c.printable(ark, chunk, dir, ""))
	} else {
		entries, err := iosrc.ReadDir(context.TODO(), dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error listing directory: %v", err)
			return
		}
		for _, e := range entries {
			fmt.Println(c.printable(ark, chunk, dir, e.Name()))
		}
	}
}

func (c *Command) printable(ark *archive.Archive, chunk archive.Chunk, zardir iosrc.URI, objPath string) string {
	if c.showRanges {
		return path.Join(chunk.Range(ark), objPath)
	}
	if c.relativePaths {
		return strings.TrimSuffix(ark.DataPath.RelPath(zardir.AppendPath(objPath)), "/")
	}
	return zardir.AppendPath(objPath).String()
}
