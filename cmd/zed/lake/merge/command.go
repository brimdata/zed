package merge

import (
	"errors"
	"flag"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Merge = &charm.Spec{
	Name:  "merge",
	Usage: "merge id [id ...]",
	Short: "merge a sequence of commits or objects into a single commit",
	Long: `
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Merge)
}

type Command struct {
	*zedlake.Command
	lakeFlags zedlake.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	return errors.New("issue #2537")
}
