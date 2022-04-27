package index

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var create = &charm.Spec{
	Name:  "create",
	Usage: "create [options] rule-name pattern",
	Short: "create an index rule for a lake",
	New:   newCreate,
}

type createCommand struct {
	*Command
	framesize   int
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
}

func newCreate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &createCommand{Command: parent.(*Command)}
	f.IntVar(&c.framesize, "framesize", 32*1024, "minimum frame size used in microindex file")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *createCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.procFlags, &c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) < 2 {
		return errors.New("a rule name and at least one index pattern must be specified")
	}
	ruleName := args[0]
	args = args[1:]
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	rules, err := c.parseIndexRules(ctx, lake, ruleName, args)
	if err != nil {
		return err
	}
	if err := lake.AddIndexRules(ctx, rules); err != nil {
		return err
	}
	if !c.Quiet {
		w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
		if err != nil {
			return err
		}
		query := fmt.Sprintf("from :index_rules | name == '%s'", ruleName)
		q, err := lake.Query(ctx, nil, query)
		if err != nil {
			w.Close()
			return err
		}
		defer q.Close()
		err = zio.Copy(w, q)
		if err2 := w.Close(); err == nil {
			err = err2
		}
		return err
	}
	return err
}

func (c *createCommand) parseIndexRules(ctx context.Context, lake api.Interface, ruleName string, args []string) ([]index.Rule, error) {
	var rules []index.Rule
	for len(args) > 0 {
		rest, rule, err := parseRule(args, ruleName)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
		args = rest
	}
	return rules, nil
}

func parseRule(args []string, ruleName string) ([]string, index.Rule, error) {
	switch args[0] {
	case "field":
		if len(args) < 2 {
			return nil, nil, errors.New("field index rule requires field(s) argument")
		}
		rule := index.NewFieldRule(ruleName, args[1])
		return args[2:], rule, nil
	case "type":
		if len(args) < 2 {
			return nil, nil, errors.New("type index rule requires type argument")
		}
		typ, err := zson.ParseType(zed.NewContext(), args[1])
		if err != nil {
			return nil, nil, err
		}
		rule := index.NewTypeRule(ruleName, typ)
		return args[2:], rule, nil
	case "agg":
		if len(args) < 2 {
			return nil, nil, errors.New("agg index rule requires a script argument")
		}
		script := args[1]
		rule, err := index.NewAggRule(ruleName, script)
		return args[2:], rule, err
	default:
		return nil, nil, fmt.Errorf("unknown index rule type: %q", args[0])
	}
}
