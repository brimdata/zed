package rm

import (
	"errors"
	"flag"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Vacate = &charm.Spec{
	Name:  "vacate",
	Usage: "vacate [options] commit",
	Short: "compact a pool's commit history by squashing old commit objects",
	Long: `
The vacate command compacts the commit history by squashing all of the commit
objects in the history up to the indicated commit and removing the old commits.
No other commit objects in the pool may point at any of the squashed commits.
In particular, no branch may point to any commit that would be deleted.

The branch history may contain pointers to old commit objects, but any attempt
to access them will fail as the underlying commit history will be no longer available.

DANGER ZONE.
There is no prompting or second chances here so use carefully.
Once the pool's commit history has been squashed and old commits is deleted,
they cannot be recovered.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Vacate)
}

type Command struct {
	lake zedlake.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	return errors.New("issue #2545")
}
