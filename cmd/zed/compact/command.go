package compact

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/commitflags"
	"github.com/brimdata/zed/cli/poolflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "compact",
	Usage: "compact id id [id ...]",
	Short: "compact data objects on a pool branch",
	Long: `
The compact command takes a list of data object IDs, writes the values
in those objects to a sequence of new, non-overlapping objects, and creates
a commit on HEAD replacing the old objects with the new ones.`,
	New: New,
}

type Command struct {
	*root.Command
	commitFlags  commitflags.Flags
	poolFlags    poolflags.Flags
	writeVectors bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.commitFlags.SetFlags(f)
	c.poolFlags.SetFlags(f)
	f.BoolVar(&c.writeVectors, "vectors", false, "write vectors for compacted objects")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	ids, err := lakeparse.ParseIDs(args)
	if err != nil {
		return err
	}
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.poolFlags.HEAD()
	if err != nil {
		return err
	}
	commit, err := lake.Compact(ctx, head.Pool, head.Branch, ids, c.writeVectors, c.commitFlags.CommitMessage())
	if err == nil && !c.LakeFlags.Quiet {
		fmt.Printf("%s compaction committed\n", commit)
	}
	return err
}
