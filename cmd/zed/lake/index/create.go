package index

import (
	"context"
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/zson"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [options] pattern [ ...pattern ]",
	Short: "create index rule for a lake pool",
	Long: `
TBD: update this help: Issue #2532

"zar index create" creates index files in a zar archive using one or more indexing
rules.

A pattern is either a field name or a ":" followed by a zng type name.
For example, to create two indexes, one on the field id.orig_h, and one on all
fields of type uint16, you would run:

	zar index create -R /path/to/logs id.orig_h :uint16

Each pattern results in a separate microindex file for each log file found.

For custom indexes, zql can be used instead of a pattern. This
requires specifying the key and output file name. For example:

       zar index create -k id.orig_h -o custom -z "count() by _path, id.orig_h | sort id.orig_h"
`,
	New: NewCreate,
}

type CreateCommand struct {
	lake   *zedlake.Command
	commit bool
	zedlake.CommitFlags
	framesize   int
	keys        string
	name        string
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	zed         string
}

func NewCreate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &CreateCommand{lake: parent.(*Command).Command}
	f.BoolVar(&c.commit, "commit", false, "commit added index rule if successfully written")
	f.IntVar(&c.framesize, "framesize", 32*1024, "minimum frame size used in microindex file")
	f.StringVar(&c.keys, "k", "key", "one or more comma-separated key fields (for zed script index rules only)")
	f.StringVar(&c.name, "n", "", "name of index rule (for zed script index rules only)")
	f.StringVar(&c.zed, "zed", "", "zed script for rule")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *CreateCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.procFlags, &c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) == 0 && c.zed == "" {
		return errors.New("one or more index rule patterns must be specified")
	}

	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}

	root, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}

	rules, err := c.createXRules(ctx, root, args)
	if err != nil {
		return err
	}

	if !c.lake.Quiet {
		w, err := c.outputFlags.Open(ctx)
		if err != nil {
			return err
		}

		m := zson.NewZNGMarshaler()
		m.Decorate(zson.StyleSimple)
		for _, rule := range rules {
			rec, err := m.MarshalRecord(rule)
			if err != nil {
				return err
			}
			if err := w.Write(rec); err != nil {
				return err
			}
		}
		return w.Close()
	}
	return err
}

func (c *CreateCommand) createXRules(ctx context.Context, root *lake.Root, args []string) ([]index.Rule, error) {
	var rules []index.Rule
	if c.zed != "" {
		rule, err := index.NewZedRule(c.zed, c.name, field.DottedList(c.keys))
		if err != nil {
			return nil, err
		}
		rule.Framesize = c.framesize
		rules = append(rules, rule)
	}

	for _, pattern := range args {
		rule, err := index.ParseRule(pattern)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, root.AddXRules(ctx, rules)
}
