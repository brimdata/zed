package status

import (
	"flag"
	"fmt"
	"io"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
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
	ids, err := zedlake.ParseIDs(args)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		// Show all of staging.
		ids, err = pool.ListStagedCommits(ctx)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			if !c.lake.Flags.Quiet {
				fmt.Println("staging area empty")
			}
			return nil
		}
	}
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		w := zngio.NewWriter(pipeWriter, zngio.WriterOpts{})
		pool.ScanStaging(ctx, w, ids)
		w.Close()
	}()
	r := zngio.NewReader(pipeReader, zson.NewContext())
	return zedlake.CopyToOutput(ctx, c.outputFlags, r)
}
