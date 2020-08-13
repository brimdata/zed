package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"runtime/pprof"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/pkg/s3io"
	"github.com/brimsec/zq/pkg/signalctx"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/ingest"
	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
	"golang.org/x/crypto/ssh/terminal"
)

// Version is set via the Go linker.
var version = "unknown"

var Zq = &charm.Spec{
	Name:        "zq",
	Usage:       "zq [ options ] [ zql ] file [ file ... ]",
	Short:       "command line logs processor",
	HiddenFlags: "cpuprofile,memprofile,pathregexp",
	Long: `
zq is a command-line tool for processing logs.  It applies boolean logic
to filter each log value, optionally computes analytics and transformations,
and writes the output to one or more files or standard output.

zq must be run with at least one input file specified.  As with awk, standard input
can be specified with a "-" in the place of the file name.  Output is sent to
standard output unless a -o or -d argument is provided, in which case output is
sent to the indicated file comforming to the type implied by the extension (unless
-f explicitly indicates the output type).

Supported input formats include binary and text zng, NDJSON, and
the Zeek log format.  Supported output formats include
all the input formats along with text and tabular formats.

The input file format is inferred from the data.  If multiple files are
specified, each file format is determined independently so you can mix and
match input types.  If multiple files are concatenated into a stream and
presented as standard input, the files must all be of the same type as the
beginning of stream will determine the format.

The output format is text zng by default, but can be overridden with -f.

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
	zctx            *resolver.Context
	dir             string
	jsonTypePath    string
	jsonPathRegexp  string
	jsonTypeConfig  *ndjsonio.TypeConfig
	outputFile      string
	verbose         bool
	stats           bool
	quiet           bool
	showVersion     bool
	stopErr         bool
	forceBinary     bool
	sortMemMaxBytes int
	textShortcut    bool
	cpuprofile      string
	memprofile      string
	cleanupFns      []func()
	ReaderFlags     zio.ReaderFlags
	WriterFlags     zio.WriterFlags
}

func New(f *flag.FlagSet) (charm.Command, error) {
	c := &Command{zctx: resolver.NewContext()}

	c.jsonPathRegexp = ingest.DefaultJSONPathRegexp

	// Flags added for writers are -f, -T, -F, -E, -U, and -b
	c.WriterFlags.SetFlags(f)

	// Flags added for readers are -i XXX json
	c.ReaderFlags.SetFlags(f)

	f.StringVar(&c.dir, "d", "", "directory for output data files")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.StringVar(&c.jsonTypePath, "j", "", "path to json types file")
	f.StringVar(&c.jsonPathRegexp, "pathregexp", c.jsonPathRegexp, "regexp for extracting _path from json log name (when -inferpath=true)")
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.BoolVar(&c.stats, "S", false, "display search stats on stderr")
	f.BoolVar(&c.quiet, "q", false, "don't display zql warnings")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.IntVar(&c.sortMemMaxBytes, "sortmem", proc.SortMemMaxBytes, "maximum memory used by sort, in bytes")
	f.BoolVar(&c.showVersion, "version", false, "print version and exit")
	f.BoolVar(&c.textShortcut, "t", false, "use format tzng independent of -f option")
	f.BoolVar(&c.forceBinary, "B", false, "allow binary zng be sent to a terminal output")
	f.StringVar(&c.cpuprofile, "cpuprofile", "", "write cpu profile to `file`")
	f.StringVar(&c.memprofile, "memprofile", "", "write memory profile to `file`")
	return c, nil
}

func fileExists(path string) bool {
	if path == "-" {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
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
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil, fmt.Errorf("%s: unmarshaling error: %s", c.jsonTypePath, err)
	}
	if err := tc.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %s", c.jsonTypePath, err)
	}
	return &tc, nil
}

func isTerminal(f *os.File) bool {
	return terminal.IsTerminal(int(f.Fd()))
}

func (c *Command) Run(args []string) error {
	defer c.runCleanup()
	if c.showVersion {
		return c.printVersion()
	}
	if len(args) == 0 {
		return Zq.Exec(c, []string{"help"})
	}
	if c.cpuprofile != "" {
		c.runCPUProfile()
	}
	if c.memprofile != "" {
		c.cleanup(c.runMemProfile)
	}
	if c.textShortcut {
		c.WriterFlags.Format = "tzng"
	}
	if c.outputFile == "" && c.WriterFlags.Format == "zng" && isTerminal(os.Stdout) && !c.forceBinary {
		return errors.New("zq: writing binary zng data to terminal; override with -B or use -t for text.")
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
	if c.sortMemMaxBytes <= 0 {
		return errors.New("sortmem value must be greater than zero")
	}
	proc.SortMemMaxBytes = c.sortMemMaxBytes
	paths := args
	var query ast.Proc
	var err error
	if fileExists(paths[0]) || s3io.IsS3Path(paths[0]) {
		query, err = zql.ParseProc("*")
		if err != nil {
			return err
		}
	} else {
		paths = paths[1:]
		if len(paths) == 0 {
			return fmt.Errorf("file not found: %s", args[0])
		}
		query, err = zql.ParseProc(args[0])
		if err != nil {
			return fmt.Errorf("parse error: %s", err)
		}
	}
	if c.WriterFlags.Format == "types" {
		logger, err := emitter.NewTypeLogger(c.outputFile, c.verbose)
		if err != nil {
			return err
		}
		c.zctx.SetLogger(logger)
		c.WriterFlags.Format = "null"
		defer logger.Close()
	}

	readers, err := c.inputReaders(paths)
	if err != nil {
		return err
	}

	wch := make(chan string, 5)
	if !c.stopErr {
		for i, r := range readers {
			readers[i] = zbuf.NewWarningReader(r, wch)
		}
	}
	reader := zbuf.NewCombiner(readers, zbuf.CmpTimeForward)
	defer reader.Close()

	writer, err := c.openOutput()
	if err != nil {
		return err
	}
	d := driver.NewCLI(writer)
	if !c.quiet {
		d.SetWarningsWriter(os.Stderr)
	}
	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()
	if err := driver.Run(ctx, d, query, c.zctx, reader, driver.Config{
		Warnings: wch,
	}); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}

func (c *Command) errorf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
}

func (c *Command) inputReaders(paths []string) ([]zbuf.Reader, error) {
	cfg := detector.OpenConfig{
		Format:         c.ReaderFlags.Format,
		JSONTypeConfig: c.jsonTypeConfig,
		JSONPathRegex:  c.jsonPathRegexp,
	}
	var readers []zbuf.Reader
	for _, path := range paths {
		if path == "-" {
			path = detector.StdinPath
		}
		file, err := detector.OpenFile(c.zctx, path, cfg)
		if err != nil {
			err = fmt.Errorf("%s: %w", path, err)
			if c.stopErr {
				return nil, err
			}
			c.errorf("%s\n", err)
			continue
		}
		readers = append(readers, file)
	}
	return readers, nil
}

func (c *Command) openOutput() (zbuf.WriteCloser, error) {
	if c.dir != "" {
		d, err := emitter.NewDir(c.dir, c.outputFile, os.Stderr, &c.WriterFlags)
		if err != nil {
			return nil, err
		}
		return d, nil
	}
	w, err := emitter.NewFile(c.outputFile, &c.WriterFlags)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (c *Command) cleanup(f func()) {
	c.cleanupFns = append(c.cleanupFns, f)
}

func (c *Command) runCleanup() {
	for _, fn := range c.cleanupFns {
		fn()
	}
}

func (c *Command) runCPUProfile() {
	f, err := os.Create(c.cpuprofile)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	c.cleanup(pprof.StopCPUProfile)
}

func (c *Command) runMemProfile() {
	f, err := os.Create(c.memprofile)
	if err != nil {
		log.Fatal(err)
	}
	runtime.GC()
	pprof.WriteHeapProfile(f)
	f.Close()
}
