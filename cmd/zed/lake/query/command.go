package zed

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/cli/searchflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/zson"
)

var Query = &charm.Spec{
	Name:  "query",
	Usage: "query [options] zql [path...]",
	Short: "run a Zed program over a data lake",
	Long: `
"zed lake query" executes a Zed query against data in a data lake.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Query)
}

type Command struct {
	lake        *zedlake.Command
	stats       bool
	stopErr     bool
	at          string
	includes    query.Includes
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	searchFlags searchflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.StringVar(&c.at, "at", "", "commit tag or journal ID for time travel")
	f.BoolVar(&c.stats, "s", false, "print search stats to stderr on successful completion")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.Var(&c.includes, "I", "source file containing Zed query text (may be used multiple times)")
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	c.searchFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.outputFlags, &c.procFlags, &c.searchFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	query, err := query.ParseSources(args, c.includes)
	if err != nil {
		return fmt.Errorf("zed lake query: %w", err)
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	pool, err := c.lake.Flags.OpenPool(ctx)
	if err != nil {
		return err
	}
	var id journal.ID
	if c.at != "" {
		id, err = zedlake.ParseJournalID(ctx, pool, c.at)
		if err != nil {
			return fmt.Errorf("zed lake query: %w", err)
		}
	}
	msrc := lake.NewMultiSourceAt(pool, id)
	writer, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	d := driver.NewCLI(writer)
	if !c.lake.Flags.Quiet {
		d.SetWarningsWriter(os.Stderr)
	}
	err = driver.MultiRun(ctx, d, query, zson.NewContext(), msrc, driver.MultiConfig{
		Span: c.searchFlags.Span(),
	})
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		c.printStats(msrc)
	}
	return err
}

func (c *Command) printStats(msrc lake.MultiSource) {
	if c.stats {
		stats := msrc.Stats()
		w := tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', 0)
		fmt.Fprintf(w, "data opened:\t%d\n", stats.TotalBytes)
		fmt.Fprintf(w, "data read:\t%d\n", stats.ReadBytes)
		w.Flush()
	}
}
