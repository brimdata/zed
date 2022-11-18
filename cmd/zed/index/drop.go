package index

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
)

var drop = &charm.Spec{
	Name:  "drop",
	Usage: "drop [-R root] [options] id... ",
	Short: "drop rule from a lake index",
	New:   newDrop,
}

type dropCommand struct {
	*Command
}

func newDrop(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &dropCommand{Command: parent.(*Command)}, nil
}

func (c *dropCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("must specify one or more index tags")
	}
	ids, err := lakeparse.ParseIDs(args)
	if err != nil {
		return err
	}
	lake, err := c.LakeFlags.Open(ctx)
	if err != nil {
		return err
	}
	rules, err := lake.DeleteIndexRules(ctx, ids)
	if err != nil {
		return err
	}
	if !c.LakeFlags.Quiet {
		for _, rule := range rules {
			fmt.Printf("%s dropped from rule %q\n", rule.RuleID(), rule.RuleName())
		}
	}
	return nil
}
