package query

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
)

var Cmd = &charm.Spec{
	Name:  "query",
	Usage: "query [options] [zed-query]",
	Short: "run a Zed query against a data lake",
	Long: `
"zed lake query" runs a Zed query against a data lake.
`,
	New: New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	stats       bool
	stopErr     bool
	includes    Includes
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	lakeFlags   lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.BoolVar(&c.stats, "s", false, "print search stats to stderr on successful completion")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.Var(&c.includes, "I", "source file containing Zed query text (may be used multiple times)")
	c.outputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags, &c.procFlags)
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
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	writer, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	d := driver.NewCLI(writer)
	if !c.lakeFlags.Quiet {
		d.SetWarningsWriter(os.Stderr)
	}
	head, _ := c.lakeFlags.HEAD()
	stats, err := lake.Query(ctx, d, head, src, c.includes...)
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if err == nil && c.stats {
		PrintStats(stats)
	}
	return err
}

type Includes []string

func (i Includes) String() string {
	return strings.Join(i, ",")
}

func (i *Includes) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i Includes) Read() ([]string, error) {
	var srcs []string
	for _, path := range i {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		srcs = append(srcs, string(b))
	}
	return srcs, nil
}

func ParseSourcesAndInputs(paths, includes Includes) ([]string, ast.Proc, error) {
	var src string
	if len(paths) != 0 && !cli.FileExists(paths[0]) && !isURLWithKnownScheme(paths[0], "http", "https", "s3") {
		if len(paths) == 1 {
			// We don't interpret the first arg as a query if there
			// are no additional args.
			return nil, nil, fmt.Errorf("no such file: %s", paths[0])
		}
		src = paths[0]
		paths = paths[1:]
	}
	query, err := compiler.ParseProc(src, includes...)
	if err != nil {
		return nil, nil, err
	}
	return paths, query, nil
}

func isURLWithKnownScheme(path string, schemes ...string) bool {
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	for _, s := range schemes {
		if u.Scheme == s {
			return true
		}
	}
	return false
}

func PrintStats(stats zbuf.ScannerStats) {
	out, err := zson.Marshal(stats)
	if err != nil {
		out = fmt.Sprintf("error marshaling stats: %s", err)
	}
	fmt.Fprintln(os.Stderr, out)
}
