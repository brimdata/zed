package index

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/segmentio/ksuid"
)

var Apply = &charm.Spec{
	Name:  "apply",
	Usage: "apply [options] rule tag [tag ...]",
	Short: "apply index rule to one or more data objects",
	New:   NewApply,
}

type ApplyCommand struct {
	*Command
	commit bool
	ids    []ksuid.KSUID
	zedlake.CommitFlags
}

func NewApply(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &ApplyCommand{Command: parent.(*Command)}
	f.BoolVar(&c.commit, "commit", false, "commit added index objects if successfully written")
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
	tags, err := lakeflags.ParseIDs(args[1:])
	if err != nil {
		return err
	}
	pool, err := lake.LookupPool(ctx, c.lakeFlags.PoolName)
	if err != nil {
		return err
	}
	commit, err := lake.ApplyIndexRules(ctx, ruleName, pool.ID, tags)
	if err != nil {
		return err
	}
	if c.commit {
		if err := lake.Commit(ctx, pool.ID, commit, *c.CommitRequest()); err != nil {
			return err
		}
		if !c.lakeFlags.Quiet {
			fmt.Printf("%s committed\n", commit)
		}
		return nil
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s staged\n", commit)
	}
	return nil
}
