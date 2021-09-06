package index

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/segmentio/ksuid"
)

var Apply = &charm.Spec{
	Name:  "apply",
	Usage: "apply rule tag [tag ...]",
	Short: "apply index rule to one or more data objects in a branch",
	New:   NewApply,
}

type ApplyCommand struct {
	*Command
	ids []ksuid.KSUID
	zedlake.CommitFlags
}

func NewApply(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &ApplyCommand{Command: parent.(*Command)}
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *ApplyCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	if len(args) < 2 {
		return errors.New("index apply command requires rule name and one or more object IDs")
	}
	ruleName := args[0]
	tags, err := lakeparse.ParseIDs(args[1:])
	if err != nil {
		return err
	}
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	if head.Pool == "" {
		return lakeflags.ErrNoHEAD
	}
	poolID, err := lake.PoolID(ctx, head.Pool)
	if err != nil {
		return err
	}
	commit, err := lake.ApplyIndexRules(ctx, ruleName, poolID, head.Branch, tags)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s committed\n", commit)
	}
	return nil
}
