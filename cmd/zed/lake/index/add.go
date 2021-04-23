package index

import (
	"errors"
	"flag"
	"fmt"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/segmentio/ksuid"
)

var Add = &charm.Spec{
	Name:  "add",
	Usage: "add [options] [-index indexid] tag [tag ...]",
	Short: "index one or more tags",
	New:   NewAdd,
}

type AddCommand struct {
	lake   *zedlake.Command
	commit bool
	ids    []ksuid.KSUID
	zedlake.CommitFlags
}

func NewAdd(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &AddCommand{lake: parent.(*Command).Command}
	f.BoolVar(&c.commit, "commit", false, "commit added index objects if successfully written")
	f.Func("index", "id of index to apply (can be specified multiple times", func(s string) error {
		id, err := zedlake.ParseID(s)
		c.ids = append(c.ids, id)
		return err
	})
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *AddCommand) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init()
	if err != nil {
		return err
	}
	defer cleanup()

	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	root, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	indices, err := root.LookupIndices(ctx, c.ids)
	if err != nil {
		return err
	}
	tags, err := zedlake.ParseIDs(args)
	if err != nil {
		return err
	} else if len(tags) == 0 {
		return errors.New("no data or commit tags specified")
	}
	pool, err := c.lake.OpenPool(ctx)
	if err != nil {
		return err
	}
	tags, err = pool.LookupTags(ctx, tags)
	if err != nil {
		return err
	}
	commit, err := pool.Index(ctx, indices, tags)
	if err != nil {
		return err
	}
	if c.commit {
		if err := pool.Commit(ctx, commit, c.Date.Ts(), c.User, c.Message); err != nil {
			return err
		}
		if !c.lake.Quiet {
			fmt.Printf("%s committed\n", commit)
		}
		return nil
	}
	if !c.lake.Quiet {
		fmt.Printf("%s staged\n", commit)
	}
	return nil
}
