package revert

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/pkg/charm"
)

var Revert = &charm.Spec{
	Name:  "revert",
	Usage: "revert -p pool[@branch] commit",
	Short: "revert reverses an old commit",
	Long: `
The revert command reverses the actions in a commit by applying the inverse
steps in a new commit to the tip of the indicated branch.  Any data loaded
in a reverted commit remains in the lake but no longer appears in the branch.
The new commit may recursively be reverted by an additional revert operation.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Revert)
	zedapi.Cmd.Add(Revert)
}

type Command struct {
	lake      zedlake.Command
	lakeFlags lakeflags.Flags
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	c.CommitFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("commit ID must be specified")
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolName, branchName := c.lakeFlags.Branch()
	if poolName == "" {
		return errors.New("name of pool must be supplied with -p option")
	}
	poolID, err := lake.PoolID(ctx, poolName)
	if err != nil {
		return err
	}
	if _, err := parser.ParseID(branchName); err == nil {
		return errors.New("branch must be named")
	}
	commitID, err := parser.ParseID(args[0])
	if err != nil {
		return err
	}
	revertID, err := lake.Revert(ctx, poolID, branchName, commitID, c.CommitMessage())
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%q: %s reverted in %s\n", branchName, commitID, revertID)
	}
	return nil
}
