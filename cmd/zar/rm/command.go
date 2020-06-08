package rm

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Rm = &charm.Spec{
	Name:  "rm",
	Usage: "rm file",
	Short: "remove files from zar directories in an archive",
	Long: `
"zar rm" walks a zar achive and removes the file with the given name from
each zar directory.
`,
	New: New,
}

func init() {
	root.Zar.Add(Rm)
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

func fileExists(path string) bool {
	if path == "-" {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("zar rm: no file specified")
	}
	if c.root == "" {
		return errors.New("zar rm: no archive root specified with -R or ZAR_ROOT")
	}

	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}

	return archive.Walk(ark, func(zardir string) error {
		for _, name := range args {
			path := filepath.Join(zardir, name)
			if fileExists(path) {
				if err := os.Remove(path); err != nil {
					return err
				}
				fmt.Printf("%s: removed\n", path)
			} else {
				fmt.Printf("%s: not found\n", path)
			}
		}
		return nil
	})
}
