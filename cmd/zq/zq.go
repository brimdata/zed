package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mccanne/charm"
	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/emitter"
	"github.com/mccanne/zq/pkg/bufwriter"
	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/proc"
	"github.com/mccanne/zq/scanner"
	"github.com/mccanne/zq/zql"
	"go.uber.org/zap"
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

The input and output formats are either specified explicitly or derived from
file name extensions.  Supported input formats include ZSON (.zson), JSON (.json),
NDJSON (.ndjson), and Zeek log format (.log).  Supported output formats include
all the input formats along with text and tabular formats.

zq must be run with at least one input file specified.  As with awk, standard input
can be specified with a "-" in the place of the file name.  Output is sent to
standard output unless a -o or -d argument is provided, in which case output is
sent to the indicated file comforming to the type implied by the extension (unless
-f explicitly indicates the output type).

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
	dt         *resolver.Table
	format     string
	dir        string
	path       string
	outputFile string
	verbose    bool
	stats      bool
	warnings   bool
	showTypes  bool
	showFields bool
	epochDates bool
}

func New(f *flag.FlagSet) (charm.Command, error) {
	cwd, _ := os.Getwd()
	c := &Command{dt: resolver.NewTable()}
	f.StringVar(&c.format, "f", "zson", "format for output data [text,table,zeek,json,ndjson,raw,zson]")
	f.StringVar(&c.path, "p", cwd, "path for input")
	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.warnings, "W", false, "display warnings on stderr")
	f.BoolVar(&c.showTypes, "T", false, "display field types in text output")
	f.BoolVar(&c.showFields, "F", false, "display field names in text output")
	f.BoolVar(&c.epochDates, "E", false, "display epoch timestamps in text output")
	return c, nil
}

func (c *Command) compile(p ast.Proc, reader zson.Reader) (*proc.MuxOutput, error) {
	ctx := &proc.Context{
		Context:  context.Background(),
		Resolver: resolver.NewTable(),
		Logger:   zap.NewNop(),
		Warnings: make(chan string, 5),
	}
	scr := scanner.NewScanner(reader)
	leaves, err := proc.CompileProc(nil, p, ctx, scr)
	if err != nil {
		return nil, err
	}
	return proc.NewMuxOutput(ctx, leaves), nil
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

func (c *Command) Run(args []string) error {
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
		query, err = zql.ParseProc(args[0])
		if err != nil {
			return fmt.Errorf("parse error: %s", err)
		}
	}
	var reader zson.Reader
	if len(paths) > 0 {
		if reader, err = c.loadFiles(paths); err != nil {
			return err
		}
	} else {
		// XXX lookup reader based on specified input type or just
		// use a TBD zsio.Peeker to delay creation of the reader until it reads
		// a few lines and infers the right type
		reader = zsio.LookupReader("zeek", os.Stdin, c.dt)
	}
	writer, err := c.openOutput()
	if err != nil {
		return err
	}
	defer writer.Close()
	output := emitter.NewEmitter(writer)
	mux, err := c.compile(query, reader)
	if err != nil {
		return err
	}
	return output.Run(mux)
}

func extension(format string) string {
	switch format {
	case "zeek":
		return ".log"
	case "zson":
		return ".zson"
	case "ndson":
		return ".ndson"
	case "json":
		return ".json"
	default:
		return ".txt"
	}
}

func (c *Command) loadFile(path string) (zson.Reader, error) {
	if path == "-" {
		//XXX TBD: use input format flag and/or peeker
		return zsio.LookupReader("zeek", os.Stdin, c.dt), nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errInvalidFile("is a directory")
	}
	// XXX this should go away soon once we have functionality to peek at the
	// stream.
	var reader string
	switch ext := filepath.Ext(path); ext {
	case ".ndjson":
		reader = "ndjson"
	default:
		reader = "zeek"
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return zsio.LookupReader(reader, f, c.dt), nil
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
		return c.openOutputDir()
	}
	file, err := c.openOutputFile()
	if err != nil {
		return nil, err
	}
	writer := zsio.LookupWriter(c.format, bufwriter.New(file))
	if writer == nil {
		return nil, fmt.Errorf("invalid format: %s", c.format)
	}
	return writer, nil
}

func (c *Command) openOutputFile() (*os.File, error) {
	if c.outputFile == "" {
		return os.Stdout, nil
	}
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	return os.OpenFile(c.outputFile, flags, 0600)
}

func (c *Command) openOutputDir() (*emitter.Dir, error) {
	ext := extension(c.format)
	return emitter.NewDir(c.dir, c.outputFile, ext, os.Stderr)
}
