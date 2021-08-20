package index

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
)

var Drop = &charm.Spec{
	Name:  "drop",
	Usage: "drop [-R root] [options] id... ",
	Short: "drop rule from a lake index",
	New:   NewDrop,
}

type DropCommand struct {
	*Command
}

func NewDrop(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &DropCommand{Command: parent.(*Command)}, nil
}

func (c *DropCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("must specify one or more index tags")
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	ids, err := lakeflags.ParseIDs(args)
	if err != nil {
		return err
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	rules, err := lake.DeleteIndexRules(ctx, ids)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		for _, rule := range rules {
			fmt.Printf("%s dropped from rule %q\n", rule.RuleID(), rule.RuleName())
		}
	}
	return nil
}
