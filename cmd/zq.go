package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mccanne/zq/emitter"
	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/proc"
	"github.com/mccanne/zq/scanner"
	"github.com/looky-cloud/lookytalk/ast"
	"github.com/looky-cloud/lookytalk/parser"
	"github.com/mccanne/charm"
	"go.uber.org/zap"
)

type errInvalidFile string

func (reason errInvalidFile) Error() string {
	return fmt.Sprintf("invalid file %s", string(reason))
}

var Zq = &charm.Spec{
	Name:  "zq",
	Usage: "zq [options] <search> [file...]",
	Short: "command line zeek processor",
	Long:  "",
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
	reverse    bool
	stats      bool
	warnings   bool
	showTypes  bool
	showFields bool
	epochDates bool
}

func New(f *flag.FlagSet) (charm.Command, error) {
	cwd, _ := os.Getwd()
	c := &Command{dt: resolver.NewTable()}
	f.StringVar(&c.format, "f", "text", "format for output data [text,table,zeek,json,ndjson,raw]")
	f.StringVar(&c.path, "p", cwd, "path for input")
	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.reverse, "R", false, "reverse search order (from oldest to newest)")
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
		Reverse:  c.reverse,
		Warnings: make(chan string, 5),
	}
	scr := scanner.NewScanner(reader)
	leaves, err := proc.CompileProc(nil, p, ctx, scr)
	if err != nil {
		return nil, err
	}
	return proc.NewMuxOutput(ctx, leaves), nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return Zq.Exec(c, []string{"help"})
	}

	query, err := parser.ParseProc(args[0])
	if err != nil {
		return fmt.Errorf("parse error: %s", err)
	}
	// XXX c.format should really implement the flag.Value interface.
	if err := checkFormat(c.format); err != nil {
		return err
	}
	paths := args[1:]
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
	case "ndson":
		return ".ndson"
	case "json":
		return ".json"
	default:
		return ".txt"
	}
}

func (c *Command) loadFile(path string) (zson.Reader, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errInvalidFile("is a directory")
	}
	// XXX this should go away soon
	if filepath.Ext(path) != ".log" {
		return nil, errInvalidFile("does not have .log extension")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return zsio.LookupReader("zeek", f, c.dt), nil
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
	// XXX need to create writer based on output format flag
	writer := zsio.LookupWriter("zeek", file)
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

func checkFormat(f string) error {
	switch f {
	case "zson", "zeek", "ndjson", "json", "text", "table", "raw":
		return nil
	}
	return fmt.Errorf("invalid format: %s", f)
}
