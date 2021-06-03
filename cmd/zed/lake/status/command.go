package status

import (
	"flag"

	"github.com/brimdata/zed/cli/lakecli"
	"github.com/brimdata/zed/cli/outputflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
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
	lake        *zedlake.Command
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	pool, err := c.lake.Flags.OpenPool(ctx)
	if err != nil {
		return err
	}
	ids, err := lakecli.ParseIDs(args)
	if err != nil {
		return err
	}
	w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	defer w.Close()
	return pool.ScanStaging(ctx, w, ids)
}
