package status

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/outputflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
)

var Status = &charm.Spec{
	Name:  "status",
	Usage: "status [options] [ <tag> ... ]",
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
	zedapi.Cmd.Add(Status)
}

type Command struct {
	lake        zedlake.Command
	outputFlags outputflags.Flags
	lakeFlags   lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(zedlake.Command)}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	ids, err := parser.ParseIDs(args)
	if err != nil {
		return err
	}
	w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	defer w.Close()
	lk, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	pool, err := lk.LookupPool(ctx, c.lakeFlags.PoolName)
	if err != nil {
		return err
	}
	err = lk.ScanStaging(ctx, pool.ID, w, ids)
	if errors.Is(err, lake.ErrStagingEmpty) {
		fmt.Fprintln(os.Stderr, "staging area is empty")
		err = nil
	}
	return err
}
