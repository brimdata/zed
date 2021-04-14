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
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/signalctx"
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
	*zedlake.Command
	stats       bool
	stopErr     bool
	includes    query.Includes
	lakeFlags   zedlake.Flags
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	searchFlags searchflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.BoolVar(&c.stats, "s", false, "print search stats to stderr on successful completion")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.Var(&c.includes, "I", "source file containing Zed query text (may be used multiple times)")
	c.lakeFlags.SetFlags(f)
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	c.searchFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.outputFlags, &c.procFlags, &c.searchFlags); err != nil {
		return err
	}
	query, err := query.ParseSources(args, c.includes)
	if err != nil {
		return fmt.Errorf("zed lake query: %w", err)
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	msrc := lake.NewMultiSource(pool)
	writer, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	d := driver.NewCLI(writer)
	if !c.lakeFlags.Quiet {
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
