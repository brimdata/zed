package query

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var Cmd = &charm.Spec{
	Name:        "query",
	Usage:       "query [ options ] [ zed-query ] file [ file ... ]",
	Short:       "apply Zed queries to data files or streams",
	HiddenFlags: "cpuprofile,memprofile,pathregexp",
	Long: `
"zed query" is a command for searching and analyzing data using the Zed language
(including the experimental SQL subset embedded in the Zed language).
If you have installed the shortcuts, "zq" is a shortcut for the "zed query" command.

"zed query" applies boolean logic
to filter each input value, optionally computes analytics and transformations,
and writes its output to one or more files or standard output.

"zed query" must be run with at least one input file specified.  Input files can
be file system paths; "-" for standard input; or HTTP, HTTPS, or S3 URLs.
Output is sent to standard output unless a -o or -d argument is provided, in
which case output is sent to the indicated file comforming to the type implied
by the extension (unless -f explicitly indicates the output type).

Supported input formats include CSV, JSON, NDJSON, Parquet,
ZSON, ZNG, ZST, and Zeek TSV.  Supported output formats include
all the input formats along with text and tabular formats.

For most formats, the input file format is inferred from the data.  If multiple files are
specified, each file format is determined independently so you can mix and
match input types.  If multiple files are concatenated into a stream and
presented as standard input, the files must all be of the same type as the
beginning of stream will determine the format.

The output format is binary ZNG by default, but can be overridden with -f.

After the options, a Zed "query" string may be specified as a
single argument conforming to the Zed language syntax;
i.e., it should be quoted as a single string in the shell.

If the first argument is a path to a valid file rather than a Zed query,
then the Zed query is assumed to be "*", i.e., match and output all
of the input.  If the first argument is both a valid Zed query
and an existing file, then the file overrides.

The Zed query text may include files using -I, which is particularly
convenient when a large, complex query spans multiple lines.  In this case,
these Zed files are concatenated together along with the command-line Zed query text
in the order appearing on the command-line.

See the Zed source repository for more information:

https://github.com/brimdata/zed
`,
	New: New,
}

type Command struct {
	*root.Command
	verbose     bool
	stats       bool
	quiet       bool
	stopErr     bool
	includes    Includes
	inputFlags  inputflags.Flags
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.outputFlags.SetFlags(f)
	c.inputFlags.SetFlags(f, false)
	c.procFlags.SetFlags(f)
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.quiet, "q", false, "don't display warnings")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.Var(&c.includes, "I", "source file containing Zed query text (may be used multiple times)")
	return c, nil
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

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags, &c.inputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 && len(c.includes) == 0 {
		return charm.NeedHelp
	}
	paths, query, err := ParseSourcesAndInputs(args, c.includes)
	if err != nil {
		return fmt.Errorf("zq: %w", err)
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	zctx := zed.NewContext()
	local := storage.NewLocalEngine()
	readers, err := c.inputFlags.Open(ctx, zctx, local, paths, c.stopErr)
	if err != nil {
		return err
	}
	defer zio.CloseReaders(readers)
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	d := driver.NewCLI(writer)
	if !c.quiet {
		d.SetWarningsWriter(os.Stderr)
	}
	if !c.stopErr {
		for i, r := range readers {
			readers[i] = zio.NewWarningReader(r, d)
		}
	}
	adaptor := cli.NewFileAdaptor(local)
	var stats zbuf.ScannerStats
	if isJoin(query) {
		stats, err = driver.RunJoinWithFileSystem(ctx, d, query, zctx, readers, adaptor)
	} else {
		var zr zio.Reader
		if len(readers) > 0 {
			zr = zio.ConcatReader(readers...)
		}
		stats, err = driver.RunWithFileSystem(ctx, d, query, zctx, zr, adaptor)
	}
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if c.stats {
		PrintStats(stats)
	}
	return err
}

func isJoin(p ast.Proc) bool {
	seq, ok := p.(*ast.Sequential)
	if !ok || len(seq.Procs) == 0 {
		return false
	}
	_, ok = seq.Procs[0].(*ast.Join)
	return ok
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
