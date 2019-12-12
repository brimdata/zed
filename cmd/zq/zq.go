package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mccanne/charm"
	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/driver"
	"github.com/mccanne/zq/emitter"
	"github.com/mccanne/zq/filter"
	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zsio/detector"
	"github.com/mccanne/zq/pkg/zsio/text"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/proc"
	"github.com/mccanne/zq/scanner"
	"github.com/mccanne/zq/zql"
)

// Version is set via the Go linker.
var (
	version = "unknown"
)

type errInvalidFile string

func (reason errInvalidFile) Error() string {
	return fmt.Sprintf("invalid file %s", string(reason))
}

var Zq = &charm.Spec{
	Name:  "zq",
	Usage: "zq [ options ] [ zql ] file [ file ... ]",
	Short: "command line logs processor",
	Long: `
zq is a command-line tool for processing logs.  It applies boolean logic
to filter each log value, optionally computes analytics and transformations,
and writes the output to one or more files or standard output.

zq must be run with at least one input file specified.  As with awk, standard input
can be specified with a "-" in the place of the file name.  Output is sent to
standard output unless a -o or -d argument is provided, in which case output is
sent to the indicated file comforming to the type implied by the extension (unless
-f explicitly indicates the output type).

Supported input formats include ZSON (.zson), NDJSON (.ndjson), and
the Zeek log format (.log).  Supported output formats include
all the input formats along with text and tabular formats.

The input file format is inferred from the data.  If multiple files are
specified, each file format is determined independently so you can mix and
match input types.  If multiple files are concatenated into a stream and
presented as standard input, the files must all be of the same type as the
beginning of stream will determine the format.

The output format is, by default, zson but can be overridden with -f.

After the options, the query may be specified as a
single argument conforming with ZQL syntax, i.e., it should be quoted as
a single string in the shell.
If the first argument is a path to a valid file rather than a ZQL query,
then the ZQL expression is assumed to be "*", i.e., match and output all
of the input.  If the first argument is both valid ZQL and an existing file,
then the file overrides.

Further details and examples for the matching and analytics syntax are described at
http://github.com/mccanne/pkg/zql/TBD.
`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Zq.Add(charm.Help)
}

type Command struct {
	dt          *resolver.Table
	ifmt        string
	ofmt        string
	dir         string
	path        string
	outputFile  string
	verbose     bool
	stats       bool
	warnings    bool
	showVersion bool
	text.Config
}

func New(f *flag.FlagSet) (charm.Command, error) {
	cwd, _ := os.Getwd()
	c := &Command{dt: resolver.NewTable()}
	f.StringVar(&c.ifmt, "i", "auto", "format of input data [auto,bzson,ndjson,zeek,zjson,zson]")
	f.StringVar(&c.ofmt, "f", "zson", "format for output data [text,table,zeek,ndjson,bzson,zson]")
	f.StringVar(&c.path, "p", cwd, "path for input")
	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.warnings, "W", false, "display warnings on stderr")
	f.BoolVar(&c.ShowTypes, "T", false, "display field types in text output")
	f.BoolVar(&c.ShowFields, "F", false, "display field names in text output")
	f.BoolVar(&c.EpochDates, "E", false, "display epoch timestamps in text output")
	f.BoolVar(&c.showVersion, "version", false, "print version and exit")
	return c, nil
}

func liftFilter(p ast.Proc) (*ast.FilterProc, ast.Proc) {
	if fp, ok := p.(*ast.FilterProc); ok {
		pass := &ast.PassProc{
			Node: ast.Node{"PassProc"},
		}
		return fp, pass
	}
	seq, ok := p.(*ast.SequentialProc)
	if ok && len(seq.Procs) > 0 {
		if fp, ok := seq.Procs[0].(*ast.FilterProc); ok {
			rest := &ast.SequentialProc{
				Node:  ast.Node{"SequentialProc"},
				Procs: seq.Procs[1:],
			}
			return fp, rest
		}
	}
	return nil, nil
}

func (c *Command) compile(program ast.Proc, reader zson.Reader) (*proc.MuxOutput, error) {
	// Try to move the filter into the scanner so we can throw
	// out unmatched records without copying their contents in the
	// case of readers (like zsio raw.Reader) that create volatile
	// records that are kepted by the scanner only if matched.
	// For other readers, it certainly doesn't hurt to do this.
	var f filter.Filter
	filterProc, rest := liftFilter(program)
	if filterProc != nil {
		var err error
		f, err = filter.Compile(filterProc.Filter)
		if err != nil {
			return nil, err
		}
		program = rest
	}
	input := scanner.NewScanner(reader, f)
	return driver.Compile(program, input)
}

func fileExists(path string) bool {
	if path == "-" {
		return true
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (c *Command) printVersion() error {
	fmt.Fprintf(os.Stdout, "Version: %s\n", version)
	return nil
}

func (c *Command) Run(args []string) error {
	if c.showVersion {
		return c.printVersion()
	}
	if len(args) == 0 {
		return Zq.Exec(c, []string{"help"})
	}
	paths := args
	var query ast.Proc
	var err error
	if fileExists(args[0]) {
		query, err = zql.ParseProc("*")
		if err != nil {
			return err
		}
	} else {
		paths = args[1:]
		if len(paths) == 0 {
			return Zq.Exec(c, []string{"help"})
		}
		query, err = zql.ParseProc(args[0])
		if err != nil {
			return fmt.Errorf("parse error: %s", err)
		}
	}
	var reader zson.Reader
	if reader, err = c.loadFiles(paths); err != nil {
		return err
	}
	writer, err := c.openOutput()
	if err != nil {
		return err
	}
	defer writer.Close()
	mux, err := c.compile(query, reader)
	if err != nil {
		return err
	}
	output := driver.New(writer)
	if c.warnings {
		output.SetWarningsWriter(os.Stderr)
	}
	return output.Run(mux)
}

func (c *Command) loadFile(path string) (zson.Reader, error) {
	var f *os.File
	if path == "-" {
		f = os.Stdin
	} else {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, errInvalidFile("is a directory")
		}
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}

	switch c.ifmt {
	case "auto":
		return detector.NewReader(f, c.dt)
	case "ndjson", "bzson", "zeek", "zjson", "zson":
		return zsio.LookupReader(c.ifmt, f, c.dt), nil
	default:
		return nil, fmt.Errorf("unknown input format %s", c.ifmt)
	}
}

func (c *Command) errorf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

func (c *Command) loadFiles(paths []string) (zson.Reader, error) {
	var readers []zson.Reader
	for _, path := range paths {
		r, err := c.loadFile(path)
		if err != nil {
			if _, ok := err.(errInvalidFile); ok {
				c.errorf("skipping file: %s\n", err)
				continue
			}
			return nil, err
		}
		readers = append(readers, r)
	}
	if len(readers) == 1 {
		return readers[0], nil
	}
	return scanner.NewCombiner(readers), nil
}

func (c *Command) openOutput() (zson.WriteCloser, error) {
	if c.dir != "" {
		d, err := emitter.NewDir(c.dir, c.outputFile, c.ofmt, os.Stderr, &c.Config)
		if err != nil {
			return nil, err
		}
		return d, nil
	}
	w, err := emitter.NewFile(c.outputFile, c.ofmt, &c.Config)
	if err != nil {
		return nil, err
	}
	return w, nil
}
