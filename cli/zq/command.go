package zq

import (
	"flag"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/cli/queryflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zfmt"
	"github.com/brimdata/zed/zio"
)

var Cmd = &charm.Spec{
	Name:        "zq",
	Usage:       "zq [ options ] [ zed-query ] file [ file ... ]",
	Short:       "process data with Zed queries",
	HiddenFlags: "cpuprofile,memprofile,pathregexp",
	Long: `
"zq" is a command-line tool for processing data in diverse input formats,
providing search, analytics, and extensive transormations using
the Zed query language.
A query typically applies Boolean logic or keyword search to filter
the input and then transforms or analyzes the filtered stream.
Output is written to one or more files or to standard output.

A Zed query is comprised of one or more operators interconnected
into a pipeline using the Unix pipe character "|".
See https://github.com/brimdata/zed/tree/main/docs/language
for details.

Supported input formats include CSV, JSON, NDJSON, Parquet,
ZSON, ZNG, ZST, and Zeek TSV.  Supported output formats include
all the input formats along with a SQL-like table format.

"zq" must be run with at least one input.  Input files can
be file system paths; "-" for standard input; or HTTP, HTTPS, or S3 URLs.
For most types of data, the input format is automatically detected.
If multiple files are specified, each file format is determined independently
so you can mix and match input types.  If multiple files are concatenated
into a stream and presented as standard input, the files must all be of the
same type as the beginning of stream will determine the format.

Output is sent to standard output unless an output file is specified with -o.
Some output formats like Parquet are based on schemas and require all
data in the output to conform to the same schema.  To handle this, you can
either fuse the data into a union of all the record types present
(presuming all the output values are records) or you can specify the -split
flag to indicate a destination directory for separate output files for each
output type.  This flag may be used in combination with -o, which
provides the prefix for the file path, e.g.,

  zq -f parquet -split out -o example-output input.zng

When writing to stdout and stdout is a terminal, the default output format is ZSON.
Otherwise, the default format is binary ZNG.  In either case, the default
may be overridden with -f, -z, or -Z.

After the options, a Zed "query" string may be specified as a
single argument conforming to the Zed language syntax;
i.e., it should be quoted as a single string in the shell.

If the first argument is a path to a valid file rather than a Zed query,
then the Zed query is assumed to be "*", i.e., match and output all
of the input.  If the first argument is both a valid Zed query
and an existing file, then the file overrides.

The Zed query text may include source files using -I, which is particularly
convenient when a large, complex query spans multiple lines.  In this case,
these source files are concatenated together along with the command-line query text
in the order appearing on the command line.

The "zq" engine processes data natively in Zed so if you intend to run
many queries over the same data, you will see substantial performance gains
by converting your data to the efficient binary form of Zed called ZNG, e.g.,

  zq -f zng input.json > fast.zng
  zq <query> fast.zng
  ...

Please see https://github.com/brimdata/zq and
https://github.com/brimdata/zed for more information.
`,
	New: New,
}

type Command struct {
	*root.Command
	canon       bool
	quiet       bool
	stopErr     bool
	queryFlags  queryflags.Flags
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
	c.queryFlags.SetFlags(f)
	f.BoolVar(&c.canon, "C", false, "display AST in Zed canonical format")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.BoolVar(&c.quiet, "q", false, "don't display warnings")
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags, &c.inputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 && len(c.queryFlags.Includes) == 0 {
		return charm.NeedHelp
	}
	if c.canon && len(args) == 1 {
		// Prevent ParseSourcesAndInputs from treating args[0] as a path.
		args = append(args, "-")
	}
	paths, flowgraph, null, err := c.queryFlags.ParseSourcesAndInputs(args)
	if err != nil {
		return fmt.Errorf("zq: %w", err)
	}
	if c.canon {
		fmt.Println(zfmt.AST(flowgraph))
		return nil
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	zctx := zed.NewContext()
	local := storage.NewLocalEngine()
	var readers []zio.Reader
	if null {
		readers = []zio.Reader{zbuf.NewArray([]zed.Value{*zed.Null})}
	} else {
		readers, err = c.inputFlags.Open(ctx, zctx, local, paths, c.stopErr)
		if err != nil {
			return err
		}
	}
	defer zio.CloseReaders(readers)
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	query, err := runtime.NewQueryOnFileSystem(ctx, zctx, flowgraph, readers, cli.NewFileAdaptor(local))
	if err != nil {
		return err
	}
	defer query.Pull(true)
	err = zio.Copy(writer, zbuf.NoControl(query.AsReader()))
	if closeErr := writer.Close(); err == nil {
		err = closeErr
	}
	c.queryFlags.PrintStats(query.Progress())
	return err
}
