package ls

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [-d <dir>]",
	Short: "list the zar directories in an archive",
	Long: `
"zar ls" descends the directory given by the -d option and prints out
the path of each zar directory.  TBD: In the future, this command could
display a detailed summary of the context each zar directory.
`,
	New: New,
}

func init() {
	root.Zar.Add(Ls)
}

type Command struct {
	*root.Command
	dir     string
	pattern string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("zar ls: no directory specified")
	}
	for _, dir := range args {
		err := archive.Walk(dir, func(zardir string) error {
			fmt.Println(zardir)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
