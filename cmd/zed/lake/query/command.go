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
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/signalctx"
	"github.com/brimdata/zed/zng/resolver"
)

var Query = &charm.Spec{
	Name:  "query",
	Usage: "query [-R root] [options] zql [file...]",
	Short: "execute ZQL against all archive directories",
	Long: `
"zed lake query" executes a zed query against one or more files from all the directories
of a zed lake, generating a single result stream. By default, the chunk file in
each directory is used, but one or more files may be specified. The special file
name "_" refers to the chunk file itself, and other names are interpreted
relative to each chunk's directory.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Query)
}

type Command struct {
	*zedlake.Command
	quiet       bool
	root        string
	stats       bool
	stopErr     bool
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	searchFlags searchflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.BoolVar(&c.quiet, "q", false, "don't display zql warnings")
	f.StringVar(&c.root, "R", os.Getenv("ZED_LAKE_ROOT"), "root directory of zar archive to walk")
	f.BoolVar(&c.stats, "s", false, "print search stats to stderr on successful completion")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
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

	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}

	query, err := compiler.ParseProc(args[0])
	if err != nil {
		return err
	}

	lk, err := lake.OpenLake(c.root, nil)
	if err != nil {
		return err
	}
	msrc := lake.NewMultiSource(lk, args[1:])

	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()

	writer, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	d := driver.NewCLI(writer)
	if !c.quiet {
		d.SetWarningsWriter(os.Stderr)
	}
	err = driver.MultiRun(ctx, d, query, resolver.NewContext(), msrc, driver.MultiConfig{
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
		fmt.Fprintf(w, "data opened:\t%d\n", stats.ChunksOpenedBytes)
		fmt.Fprintf(w, "data read:\t%d\n", stats.ChunksReadBytes)
		w.Flush()
	}
}
