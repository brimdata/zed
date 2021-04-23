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
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [options] pattern",
	Short: "create an index for a lake",
	New:   NewCreate,
}

type CreateCommand struct {
	lake        *zedlake.Command
	framesize   int
	keys        string
	name        string
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	zed         string
}

func NewCreate(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &CreateCommand{lake: parent.(*Command).Command}
	f.IntVar(&c.framesize, "framesize", 32*1024, "minimum frame size used in microindex file")
	f.StringVar(&c.keys, "k", "key", "one or more comma-separated key fields (for zed script indices only)")
	f.StringVar(&c.name, "n", "", "name of index (for zed script indices only)")
	f.StringVar(&c.zed, "zed", "", "zed script for index")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *CreateCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.procFlags, &c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 && c.zed == "" {
		return errors.New("at least one index pattern must be specified")
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	root, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	rules, err := c.createIndices(ctx, root, args)
	if err != nil {
		return err
	}
	if !c.lake.Quiet {
		w, err := c.outputFlags.Open(ctx)
		if err != nil {
			return err
		}
		if err := root.ScanIndex(ctx, w, rules.IDs()); err != nil {
			return err
		}
		return w.Close()
	}
	return err
}

func (c *CreateCommand) createIndices(ctx context.Context, root *lake.Root, args []string) (index.Indices, error) {
	var rules []index.Index
	if c.zed != "" {
		rule, err := index.NewZedIndex(c.zed, c.name, field.DottedList(c.keys))
		if err != nil {
			return nil, err
		}
		rule.Framesize = c.framesize
		rules = append(rules, rule)
	}

	for _, pattern := range args {
		rule, err := index.ParseIndex(pattern)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, root.AddIndex(ctx, rules)
}
