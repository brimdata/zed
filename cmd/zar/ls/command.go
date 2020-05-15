package ls

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [options] [pattern]",
	Short: "list the zar directories in an archive",
	Long: `
"zar ls" walks an archive's directories and prints out
the path of each zar directory contained with those top-level directories.
TBD: In the future, this command could
display a detailed summary of the context each zar directory.
`,
	New: New,
}

func init() {
	root.Zar.Add(Ls)
}

type Command struct {
	*root.Command
	root  string
	lflag bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.BoolVar(&c.lflag, "l", false, "long form")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 1 {
		return errors.New("zar ls: too many arguments")
	}

	ark, err := archive.OpenArchive(c.root)
	if err != nil {
		return err
	}

	var pattern string
	if len(args) == 1 {
		pattern = args[0]
	}
	return archive.Walk(ark, func(zardir string) error {
		printDir(zardir, pattern, c.lflag)
		return nil
	})
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func printDir(dir, pattern string, lflag bool) {
	if pattern != "" {
		path := filepath.Join(dir, pattern)
		if fileExists(path) {
			fmt.Println(path)
		}
		return
	}
	fmt.Println(dir)
	if lflag {
		files := ls(dir)
		for _, f := range files {
			fmt.Printf("\t%s\n", f)
		}
	}
}

func ls(dir string) []string {
	var out []string
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}
	return out
}
