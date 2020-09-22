package rm

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zqe"
	"github.com/mccanne/charm"
)

var Rm = &charm.Spec{
	Name:  "rm",
	Usage: "rm [-R root] file",
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
	root          string
	relativePaths bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.relativePaths, "relative", false, "display paths relative to root")
	return c, nil
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

	return archive.Walk(context.TODO(), ark, func(zardir iosrc.URI) error {
		for _, name := range args {
			path := zardir.AppendPath(name)
			if err := iosrc.Remove(context.TODO(), path); err != nil {
				if zqe.IsNotFound(err) {
					fmt.Printf("%s: not found\n", c.printable(ark.DataPath, path))
					continue
				}
				return err
			}
			fmt.Printf("%s: removed\n", c.printable(ark.DataPath, path))
		}
		return nil
	})
}

func (c *Command) printable(root, path iosrc.URI) string {
	if c.relativePaths {
		return root.RelPath(path)
	}
	return path.String()
}
