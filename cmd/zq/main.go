package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	zedquery "github.com/brimdata/zed/cmd/zed/query"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
)

var Cmd = &charm.Spec{
	Name:        "zq",
	Usage:       "zq [ options ] [ zed-query ] file [ file ... ]",
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
	includes    zedquery.Includes
	inputFlags  inputflags.Flags
	outputFlags outputflags.Flags
	procFlags   procflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	parent, err := root.New(parent, f)
	if err != nil {
		return nil, nil
	}
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

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags, &c.inputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 && len(c.includes) == 0 {
		return charm.NeedHelp
	}
	paths, query, err := zedquery.ParseSourcesAndInputs(args, c.includes)
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
	stats, err := driver.RunWithFileSystem(ctx, d, query, zctx, readers, cli.NewFileAdaptor(local))
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	if c.stats {
		zedquery.PrintStats(stats)
	}
	return err
}

func main() {
	if err := Cmd.ExecRoot(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
