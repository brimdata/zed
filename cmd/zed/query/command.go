package query

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/s3io"
	"github.com/brimdata/zed/pkg/signalctx"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng/resolver"
)

var Cmd = &charm.Spec{
	Name:        "query",
	Usage:       "query [ options ] [ zed-query ] file [ file ... ]",
	Short:       "apply zed queries to data files or streams",
	HiddenFlags: "cpuprofile,memprofile,pathregexp",
	Long: `
"zed query" is a command for searching and analyzing data using the zed language
(including the experimental SQL subset embedded in the zed language).
If you have istalled the shortcuts, "zq" is a shortcut for the "zed query" command.

"zed query" applies boolean logic
to filter each input value, optionally computes analytics and transformations,
and writes its output to one or more files or standard output.

"zed query" must be run with at least one input file specified.  As with awk, standard input
can be specified with a "-" in the place of the file name.  Output is sent to
standard output unless a -o or -d argument is provided, in which case output is
sent to the indicated file comforming to the type implied by the extension (unless
-f explicitly indicates the output type).

Supported input formats include CSV, NDJSON, Parquet,
ZSON, ZNG, ZST, and Zeek TSV.  Supported output formats include
all the input formats along with text and tabular formats.

The input file format is inferred from the data.  If multiple files are
specified, each file format is determined independently so you can mix and
match input types.  If multiple files are concatenated into a stream and
presented as standard input, the files must all be of the same type as the
beginning of stream will determine the format.

The output format is binary ZNG by default, but can be overridden with -f.

After the options, a zed "query" string may be specified as a
single argument conforming to the zed language syntax;
i.e., it should be quoted as a single string in the shell.

If the first argument is a path to a valid file rather than a zed query,
then the zed query is assumed to be "*", i.e., match and output all
of the input.  If the first argument is both a valid zed query
and an existing file, then the file overrides.

The zed query text may include files using -I, which is particularly
convenient when a large, complex query spans multiple lines.  In this case,
these zed files are concatenated together along with the command-line zed query text
in the order appearing on the command-line.

See the zed source repository for more information:

https://github.com/brimdata/zed
`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

type Command struct {
	verbose     bool
	stats       bool
	quiet       bool
	stopErr     bool
	parallel    bool
	includes    includes
	inputFlags  inputflags.Flags
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
	cli         cli.Flags
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{}

	// Flags added for writers are -f, -T, -F, -E, -U, and -b
	c.outputFlags.SetFlags(f)

	// Flags added for readers are -i, etc
	c.inputFlags.SetFlags(f)

	c.procFlags.SetFlags(f)

	c.cli.SetFlags(f)

	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.quiet, "q", false, "don't display zql warnings")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.BoolVar(&c.parallel, "P", false, "read two or more files into parallel-input zql query")
	f.Var(&c.includes, "I", "source file containing Z query text (may be used multiple times)")
	return c, nil
}

type includes []string

func (i includes) String() string {
	return strings.Join(i, ",")
}

func (i *includes) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (c *Command) Run(args []string) error {
	defer c.cli.Cleanup()
	err := c.cli.Init(&c.outputFlags, &c.inputFlags, &c.procFlags)
	if len(args) == 0 {
		return charm.NeedHelp
	}
	if err != nil {
		return err
	}
	var srcs []string
	for _, path := range c.includes {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		srcs = append(srcs, string(b))
	}
	paths := args
	if !cli.FileExists(paths[0]) && !s3io.IsS3Path(paths[0]) {
		if len(paths) == 1 {
			// We don't interpret the first arg as a query if there
			// are no additional args.
			return fmt.Errorf("zq: no such file: %s", paths[0])
		}
		srcs = append(srcs, paths[0])
		paths = paths[1:]
	}
	zqlSrc := strings.Join(srcs, "\n")
	if zqlSrc == "" {
		zqlSrc = "*"
	}
	query, err := compiler.ParseProc(zqlSrc)
	if err != nil {
		return fmt.Errorf("zq: parse error: %w", err)
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	zctx := resolver.NewContext()
	readers, err := c.inputFlags.Open(zctx, paths, c.stopErr)
	if err != nil {
		return err
	}

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
	if !c.stopErr {
		for i, r := range readers {
			readers[i] = zbuf.NewWarningReader(r, d)
		}
	}
	defer zbuf.CloseReaders(readers)

	if c.parallel {
		if err := driver.RunParallel(ctx, d, query, zctx, readers, driver.Config{}); err != nil {
			writer.Close()
			return err
		}
	} else {
		reader, err := zbuf.MergeReadersByTsAsReader(ctx, readers, zbuf.OrderAsc)
		if err != nil {
			writer.Close()
			return err
		}
		if err := driver.Run(ctx, d, query, zctx, reader, driver.Config{}); err != nil {
			writer.Close()
			return err
		}
	}
	return writer.Close()
}
