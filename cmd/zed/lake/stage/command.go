package stage

import (
	"context"
	"errors"
	"flag"
	"fmt"

	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zqe"
	"github.com/segmentio/ksuid"
)

var Stage = &charm.Spec{
	Name:  "stage",
	Usage: "stage [-R root] [options] [staging-tag]",
	Short: "list commits in staging",
	Long: `
"zed lake stage" shows a data pool's pending commits from its staging area.
If a staging-tag (e.g., as output by "zed lake add") is given,
then details for that pending commit are displayed.

If -drop is specified, then one or more commits are required and the commit
is deleted from staging along with the underlying data that was written
into the lake.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Stage)
}

type Command struct {
	*zedlake.Command
	drop      bool
	lakeFlags zedlake.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.BoolVar(&c.drop, "drop", false, "delete specified commits from staging")
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx := context.TODO()
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.New("zed lake staging: too many arguments")
	}
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		if c.drop {
			return errors.New("no commits specified for deletion")
		}
		commits, err := pool.GetStagedCommits(ctx)
		if err != nil {
			return err
		}
		if len(commits) == 0 {
			fmt.Println("no commits in staging")
			return nil
		}
		for _, c := range commits {
			if err := printCommit(ctx, pool, c); err != nil {
				return err
			}
		}
		return nil
	}
	ids, err := zedlake.ParseIDs(args)
	if err != nil {
		return err
	}
	if c.drop {
		return errors.New("TBD: issue #2541")
	}
	return printCommits(ctx, pool, ids)
}

func printCommits(ctx context.Context, pool *lake.Pool, ids []ksuid.KSUID) error {
	for _, id := range ids {
		if err := printCommit(ctx, pool, id); err != nil {
			if zqe.IsNotFound(err) {
				err = fmt.Errorf("%s: not found", id)
			}
			return err
		}
	}
	return nil
}

func printCommit(ctx context.Context, pool *lake.Pool, id ksuid.KSUID) error {
	txn, err := pool.LoadFromStaging(ctx, id)
	if err != nil {
		return err
	}
	fmt.Printf("commit %s\n", id)
	for _, action := range txn {
		//XXX
		fmt.Printf("  segment %s\n", action)
	}
	return nil
}
