package tag

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake/api"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
)

var Cmd = &charm.Spec{
	Name:  "tag",
	Usage: "tag new-tag [base]",
	Short: "create a new tag",
	Long: `
The lake tag command creates a new tag with the indicated name.
If specified, base is either an existing branch name, tag name, or a commit ID
and provides the new tag's base. If not specified, then HEAD is assumed.

If the -d option is specified, then the tag is deleted. No data is
deleted by this operation and the deleted tag can be easily recreated by
running the tag command again with the commit ID desired.
`,
	New: New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	delete      bool
	lakeFlags   lakeflags.Flags
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.BoolVar(&c.delete, "d", false, "delete the tag instead of creating it")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.New("too many arguments")
	}
	defer cleanup()
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return c.list(ctx, lake)
	}
	tagName := args[0]

	// TBD tag from [base]
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolName := head.Pool
	if poolName == "" {
		return errors.New("a pool name must be included: pool@branch")
	}
	poolID, err := lakeparse.ParseID(poolName)
	if err != nil {
		poolID, err = lake.PoolID(ctx, poolName)
		if err != nil {
			return err
		}
	}
	parentCommit, err := lakeparse.ParseID(head.Branch)
	if err != nil {
		parentCommit, err = lake.CommitObject(ctx, poolID, head.Branch)
		if err != nil {
			return err
		}
	}
	if c.delete {
		if err := lake.RemoveTag(ctx, poolID, tagName); err != nil {
			return err
		}
		if !c.lakeFlags.Quiet {
			fmt.Printf("tag deleted: %s\n", tagName)
		}
		return nil
	}
	if err := lake.CreateTag(ctx, poolID, tagName, parentCommit); err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%q: tag created\n", tagName)
	}
	return nil
}

func (c *Command) list(ctx context.Context, lake api.Interface) error {
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	poolName := head.Pool
	if poolName == "" {
		return errors.New("must be on a checked out branch to list the tags in the same pool")
	}
	query := fmt.Sprintf("from '%s':tags", poolName)
	if c.outputFlags.Format == "lake" {
		c.outputFlags.WriterOpts.Lake.Head = head.Branch
	}
	w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	q, err := lake.Query(ctx, nil, query)
	if err != nil {
		w.Close()
		return err
	}
	err = zio.Copy(w, q)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}
