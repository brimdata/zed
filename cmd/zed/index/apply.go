package index

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
)

var apply = &charm.Spec{
	Name:  "apply",
	Usage: "apply -r rule [-r rule ...] tag [tag ...]",
	Short: "apply index rules to one or more data objects in a branch",
	New:   newApply,
}

type applyCommand struct {
	*Command
	rules []string
}

func newApply(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &applyCommand{Command: parent.(*Command)}
	f.Func("r", "name of index rule to apply; can be set multiple times", func(s string) error {
		if s == "" {
			return errors.New("rule cannot be an empty string")
		}
		c.rules = append(c.rules, s)
		return nil
	})
	return c, nil
}

func (c *applyCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	if len(args) < 1 {
		return errors.New("index apply: one or more object IDs must be provided as arguments")
	}
	if len(c.rules) == 0 {
		return errors.New("index apply: at least one index rule must be specified (use the -r flag)")
	}
	tags, err := lakeparse.ParseIDs(args)
	if err != nil {
		return err
	}
	head, err := c.LakeFlags.HEAD()
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
	commit, err := lake.ApplyIndexRules(ctx, c.rules, poolID, head.Branch, tags)
	if err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		fmt.Printf("%s committed\n", commit)
	}
	return nil
}
