package index

import (
	"context"
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [options] rule-name pattern",
	Short: "create an index rule for a lake",
	New:   NewCreate,
}

type CreateCommand struct {
	*Command
	framesize   int
	keys        string
	name        string
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	zed         string
}

func NewCreate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &CreateCommand{Command: parent.(*Command)}
	f.IntVar(&c.framesize, "framesize", 32*1024, "minimum frame size used in microindex file")
	f.StringVar(&c.keys, "k", "key", "one or more comma-separated key fields (for Zed script indices only)")
	f.StringVar(&c.name, "n", "", "name of index")
	f.StringVar(&c.zed, "zed", "", "Zed script for index")
	c.lakeFlags.SetFlags(f)
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *CreateCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init(&c.procFlags, &c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) < 2 {
		return errors.New("a rule name and at least one index pattern must be specified")
	}
	ruleName := args[0]
	args = args[1:]
	if len(args) == 0 && c.zed == "" {
		return errors.New("at least one index pattern must be specified")
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	rules, err := c.createIndices(ctx, lake, ruleName, args)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
		if err != nil {
			return err
		}
		if err := lake.ScanIndexRules(ctx, w, ruleIDs(rules)); err != nil {
			return err
		}
		return w.Close()
	}
	return err
}

func ruleIDs(rules []index.Rule) []ksuid.KSUID {
	ids := make([]ksuid.KSUID, 0, len(rules))
	for _, r := range rules {
		ids = append(ids, r.RuleID())
	}
	return ids
}

func (c *CreateCommand) createIndices(ctx context.Context, lake api.Interface, ruleName string, args []string) ([]index.Rule, error) {
	var rules []index.Rule
	if c.zed != "" {
		rule, err := index.NewZedRule(c.zed, c.name, field.DottedList(c.keys))
		if err != nil {
			return nil, err
		}
		//rule.Framesize = c.framesize
		rules = append(rules, rule)
	}
	for _, pattern := range args {
		rule, err := index.ParseRule(ruleName, pattern)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, lake.AddIndexRules(ctx, rules)
}
