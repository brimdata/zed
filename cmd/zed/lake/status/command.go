package status

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var Status = &charm.Spec{
	Name:  "status",
	Usage: "status [-R root] [options] [staging-tag]",
	Short: "list commits in staging",
	Long: `
"zed lake status" shows a data pool's pending commits from its staging area.
If a staged commit tag (e.g., as output by "zed lake add") is given,
then details for that pending commit are displayed.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Status)
}

type Command struct {
	*zedlake.Command
	lakeFlags   zedlake.Flags
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	c.lakeFlags.SetFlags(f)
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx := context.TODO()
	defer c.Cleanup()
	if err := c.Init(&c.outputFlags); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.New("zed lake status: too many arguments")
	}
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	var ids []ksuid.KSUID
	if len(args) > 0 {
		ids, err = zedlake.ParseIDs(args)
		if err != nil {
			return err
		}
	} else {
		ids, err = pool.GetStagedCommits(ctx)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			fmt.Println("no commits in staging")
			return nil
		}
	}
	txns := make([]*commit.Transaction, 0, len(ids))
	for _, id := range ids {
		txn, err := pool.LoadFromStaging(ctx, id)
		if err != nil {
			if zqe.IsNotFound(err) {
				err = fmt.Errorf("%s: not found", id)
			}
			return err
		}
		txns = append(txns, txn)
	}
	if c.outputFlags.Format == "text" {
		printCommits(txns)
		return nil
	}
	w, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	if err := marshalCommits(txns, w); err != nil {
		return err
	}
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}

func printCommits(txns []*commit.Transaction) {
	for _, txn := range txns {
		fmt.Printf("commit %s\n", txn.ID)
		for _, action := range txn.Actions {
			//XXX
			fmt.Printf("  segment %s\n", action)
		}
	}
}

func marshalCommits(txns []*commit.Transaction, w zbuf.Writer) error {
	m := zson.NewZNGMarshaler()
	for _, txn := range txns {
		rec, err := m.MarshalRecord(txn)
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}
