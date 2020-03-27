package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
)

// Version is set via the Go linker.
var version = "unknown"

// x509.14:00:00-15:00:00.log.gz (open source zeek)
// x509_20191101_14:00:00-15:00:00+0000.log.gz (corelight)
const defaultJSONPathRegexp = `([a-zA-Z0-9_]+)(?:\.|_\d{8}_)\d\d:\d\d:\d\d\-\d\d:\d\d:\d\d(?:[+\-]\d{4})?\.log(?:$|\.gz)`

var Zq = &charm.Spec{
	Name:        "zq",
	Usage:       "zq [ options ] [ zql ] file [ file ... ]",
	Short:       "command line logs processor",
	HiddenFlags: "pathregexp",
	Long: `
zq is a command-line tool for processing logs.  It applies boolean logic
to filter each log value, optionally computes analytics and transformations,
and writes the output to one or more files or standard output.

zq must be run with at least one input file specified.  As with awk, standard input
can be specified with a "-" in the place of the file name.  Output is sent to
standard output unless a -o or -d argument is provided, in which case output is
sent to the indicated file comforming to the type implied by the extension (unless
-f explicitly indicates the output type).

Supported input formats include zng (.zng), NDJSON (.ndjson), and
the Zeek log format (.log).  Supported output formats include
all the input formats along with text and tabular formats.

The input file format is inferred from the data.  If multiple files are
specified, each file format is determined independently so you can mix and
match input types.  If multiple files are concatenated into a stream and
presented as standard input, the files must all be of the same type as the
beginning of stream will determine the format.

The output format is zng by default, but can be overridden with -f.

After the options, the query may be specified as a
single argument conforming with ZQL syntax; i.e., it should be quoted as
a single string in the shell.
If the first argument is a path to a valid file rather than a ZQL query,
then the ZQL expression is assumed to be "*", i.e., match and output all
of the input.  If the first argument is both valid ZQL and an existing file,
then the file overrides.

See the zq source repository for more information:

https://github.com/brimsec/zq
`,
	New: func(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
		return New(flags)
	},
}

func init() {
	Zq.Add(charm.Help)
}

type Command struct {
	zctx           *resolver.Context
	ifmt           string
	ofmt           string
	dir            string
	path           string
	jsonTypePath   string
	jsonPathRegexp string
	jsonTypeConfig *ndjsonio.TypeConfig
	outputFile     string
	verbose        bool
	stats          bool
	quiet          bool
	showVersion    bool
	zio.Flags
}

func New(f *flag.FlagSet) (charm.Command, error) {
	cwd, _ := os.Getwd()
	c := &Command{zctx: resolver.NewContext()}

	c.jsonPathRegexp = defaultJSONPathRegexp

	f.StringVar(&c.ifmt, "i", "auto", "format of input data [auto,bzng,ndjson,zeek,zjson,zng]")
	f.StringVar(&c.ofmt, "f", "zng", "format for output data [bzng,ndjson,table,text,types,zeek,zjson,zng]")
	f.StringVar(&c.path, "p", cwd, "path for input")
	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.StringVar(&c.jsonTypePath, "j", "", "path to json types file")
	f.StringVar(&c.jsonPathRegexp, "pathregexp", c.jsonPathRegexp, "regexp for extracting _path from json log name (when -inferpath=true)")
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.quiet, "q", false, "don't display zql warnings")
	f.BoolVar(&c.ShowTypes, "T", false, "display field types in text output")
	f.BoolVar(&c.ShowFields, "F", false, "display field names in text output")
	f.BoolVar(&c.EpochDates, "E", false, "display epoch timestamps in text output")
	f.BoolVar(&c.UTF8, "U", false, "display zeek strings as UTF-8")
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

func (c *Command) compile(program ast.Proc, reader zbuf.Reader) (*proc.MuxOutput, error) {
	// Try to move the filter into the scanner so we can throw
	// out unmatched records without copying their contents in the
	// case of readers (like zio raw.Reader) that create volatile
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
	fmt.Printf("Version: %s\n", version)
	return nil
}

func (c *Command) loadJsonTypes() (*ndjsonio.TypeConfig, error) {
	data, err := ioutil.ReadFile(c.jsonTypePath)
	if err != nil {
		return nil, err
	}
	var tc ndjsonio.TypeConfig
	err = json.Unmarshal(data, &tc)
	if err != nil {
		return nil, fmt.Errorf("%s: unmarshaling error: %s", c.jsonTypePath, err)
	}
	if err = tc.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %s", c.jsonTypePath, err)
	}
	return &tc, nil
}

func (c *Command) Run(args []string) error {
	if c.showVersion {
		return c.printVersion()
	}
	if len(args) == 0 {
		return Zq.Exec(c, []string{"help"})
	}
	if _, err := regexp.Compile(c.jsonPathRegexp); err != nil {
		return err
	}
	if c.jsonTypePath != "" {
		tc, err := c.loadJsonTypes()
		if err != nil {
			return err
		}
		c.jsonTypeConfig = tc
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
			return fmt.Errorf("file not found: %s", args[0])
		}
		query, err = zql.ParseProc(args[0])
		if err != nil {
			return fmt.Errorf("parse error: %s", err)
		}
	}
	if c.ofmt == "types" {
		logger, err := emitter.NewTypeLogger(c.outputFile, c.verbose)
		if err != nil {
			return err
		}
		c.zctx.SetLogger(logger)
		c.ofmt = "null"
		defer logger.Close()
	}

	readers, err := c.inputReaders(paths)
	if err != nil {
		return err
	}

	var reader zbuf.Reader
	if len(readers) == 1 {
		reader = readers[0]
	} else {
		reader = scanner.NewCombiner(readers)
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
	if !c.quiet {
		output.SetWarningsWriter(os.Stderr)
	}
	return output.Run(mux)
}

func (c *Command) configureJSONTypeReader(ndjr *ndjsonio.Reader, filename string) error {
	var path string
	re := regexp.MustCompile(c.jsonPathRegexp)
	match := re.FindStringSubmatch(filename)
	if len(match) == 2 {
		path = match[1]
	}
	if err := ndjr.ConfigureTypes(*c.jsonTypeConfig, path); err != nil {
		return err
	}
	return nil
}

func (c *Command) errorf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

type namedReader struct {
	zbuf.Reader
	name string
}

func (r namedReader) String() string {
	return r.name
}

func (c *Command) inputReaders(paths []string) ([]zbuf.Reader, error) {
	var readers []zbuf.Reader
	for _, path := range paths {
		var zr zbuf.Reader
		var f *os.File
		if path == "-" {
			f = os.Stdin
		} else {
			var err error
			info, err := os.Stat(path)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				return nil, errors.New("is a directory")
			}
			f, err = os.Open(path)
			if err != nil {
				return nil, err
			}
		}
		r := detector.GzipReader(f)
		var err error
		if c.ifmt == "auto" {
			zr, err = detector.NewReader(r, c.zctx)
		} else {
			zr, err = detector.LookupReader(c.ifmt, r, c.zctx)
		}
		if err != nil {
			return nil, err
		}
		jr, ok := zr.(*ndjsonio.Reader)
		if ok && c.jsonTypeConfig != nil {
			if err = c.configureJSONTypeReader(jr, path); err != nil {
				return nil, err
			}
		}
		readers = append(readers, namedReader{zr, path})
	}
	return readers, nil
}

func (c *Command) openOutput() (zbuf.WriteCloser, error) {
	if c.dir != "" {
		d, err := emitter.NewDir(c.dir, c.outputFile, c.ofmt, os.Stderr, &c.Flags)
		if err != nil {
			return nil, err
		}
		return d, nil
	}
	w, err := emitter.NewFile(c.outputFile, c.ofmt, &c.Flags)
	if err != nil {
		return nil, err
	}
	return w, nil
}
