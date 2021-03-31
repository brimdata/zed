package rm

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/chunk"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zqe"
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
	zedlake.Cmd.Add(Rm)
}

type Command struct {
	*zedlake.Command
	root          string
	relativePaths bool
	showRanges    bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.BoolVar(&c.relativePaths, "relative", false, "display paths relative to root")
	f.BoolVar(&c.showRanges, "ranges", false, "display time ranges instead of paths")
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("zar rm: no file specified")
	}
	if c.root == "" {
		return errors.New("zar rm: no archive root specified with -R or ZAR_ROOT")
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}

	return lake.Walk(context.TODO(), lk, func(chunk chunk.Chunk) error {
		return c.remove(lk, chunk, args)
	})
}

func (c *Command) remove(lk *lake.Lake, chunk chunk.Chunk, names []string) error {
	for _, name := range names {
		path := chunk.ZarDir().AppendPath(name)
		if err := iosrc.Remove(context.TODO(), path); err != nil {
			if zqe.IsNotFound(err) {
				fmt.Printf("%s: not found\n", c.printable(lk, chunk, name))
				continue
			}
			return err
		}
		fmt.Printf("%s: removed\n", c.printable(lk, chunk, name))
	}
	return nil
}
func (c *Command) printable(lk *lake.Lake, chunk chunk.Chunk, objPath string) string {
	if c.showRanges {
		return path.Join(chunk.Range(), objPath)
	}
	if c.relativePaths {
		return strings.TrimSuffix(lk.DataPath.RelPath(chunk.ZarDir().AppendPath(objPath)), "/")
	}
	return chunk.ZarDir().AppendPath(objPath).String()
}
