package merge

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/segmentio/ksuid"
)

var Merge = &charm.Spec{
	Name:  "merge",
	Usage: "merge -p pool/branch [-at id]",
	Short: "merge a branch into its parent",
	Long: `
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Merge)
	zedapi.Cmd.Add(Merge)
}

type Command struct {
	lake      zedlake.Command
	at        string
	lakeFlags lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	f.StringVar(&c.at, "at", "", "commit tag on source branch to merge from")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	poolName, branchName := c.lakeFlags.Branch()
	if poolName == "" {
		return errors.New("pool name must be specified with -p")
	}
	poolID, branchID, err := lake.IDs(ctx, poolName, branchName)
	if err != nil {
		return err
	}
	var at ksuid.KSUID
	if c.at != "" {
		at, err = parser.ParseID(c.at)
		if err != nil {
			return fmt.Errorf("bad -at option: %w", err)
		}
	}
	commit, err := lake.MergeBranch(ctx, poolID, branchID, at)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("branch merged to commit %s\n", commit)
	}
	return nil
}
