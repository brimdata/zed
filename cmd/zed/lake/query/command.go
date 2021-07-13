package query

import (
	"errors"
	"flag"
	"os"

	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
)

var Query = &charm.Spec{
	Name:  "query",
	Usage: "query [options] [zed-query]",
	Short: "run a Zed query against a data lake",
	Long: `
"zed lake query" runs a Zed query against a data lake.
`,
	New: New,
}

func init() {
	zedapi.Cmd.Add(Query)
	zedlake.Cmd.Add(Query)
}

type Command struct {
	lake        *zedlake.Command
	stats       bool
	stopErr     bool
	includes    query.Includes
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.BoolVar(&c.stats, "s", false, "print search stats to stderr on successful completion")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.Var(&c.includes, "I", "source file containing Zed query text (may be used multiple times)")
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.outputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) > 1 || len(args) == 0 && len(c.includes) == 0 {
		return charm.NeedHelp
	}
	var src string
	if len(args) == 1 {
		src = args[0]
	}
	if c.lake.Flags.PoolName() != "" {
		return errors.New("zed lake query: use from operator instead of -p")
	}
	lk, err := c.lake.Flags.Open(ctx)
	if err != nil {
		return err
	}
	writer, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	d := driver.NewCLI(writer)
	if !c.lake.Flags.Quiet() {
		d.SetWarningsWriter(os.Stderr)
	}
	stats, err := lk.Query(ctx, d, src, c.includes...)
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if err == nil && c.stats {
		query.PrintStats(stats)
	}
	return err
}
