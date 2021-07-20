package rm

import (
	"errors"
	"flag"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
)

var Vacate = &charm.Spec{
	Name:  "vacate",
	Usage: "vacate [options] journal-id",
	Short: "advance the tail of a pool's commit journal and delete old data",
	Long: `
The vacate command advances the tail of a pool's commit journal so that any commits
before the new tail cannot be accessed and thus "time travel" to previous
such commits can no longer be accomplished.  Data segments that are no
longer accessible after the tail has been advanced are removed from the
underlying storage system.

The only time you should use
this is when you want to free up old data that no longer needs to be accessed.

DANGER ZONE.
There is no prompting or second chances here so use carefully.
Once the pool's tail has been advanced and old data is deleted,
it cannot be recovered.
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
