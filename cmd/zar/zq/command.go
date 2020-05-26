package zq

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
	"go.uber.org/zap"
)

var Zq = &charm.Spec{
	Name:  "zq",
	Usage: "zq [-R dir] [options] [zql] file [file...]",
	Short: "walk an archive and run zql queries",
	Long: `
"zar zq" descends the directory given by the -R option (or ZAR_ROOT env) looking for
logs with zar directories and for each such directory found, it runs
the zq logic relative to that directory and emits the results in zng format.
The file names here are relative to that directory and the special name "_" refers
to the actual log file in the parent of the zar directory.

If the root directory is not specified by either the ZAR_ROOT environemnt
variable or the -R option, then the current directory is assumed.
`,
	New: New,
}

func init() {
	root.Zar.Add(Zq)
}

type Command struct {
	*root.Command
	root           string
	jsonTypePath   string
	jsonPathRegexp string
	jsonTypeConfig *ndjsonio.TypeConfig
	outputFile     string
	stopErr        bool
	quiet          bool
	ReaderFlags    zio.ReaderFlags
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

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.BoolVar(&c.quiet, "q", false, "don't display zql warnings")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.StringVar(&c.jsonTypePath, "j", "", "path to json types file")
	f.StringVar(&c.jsonPathRegexp, "pathregexp", c.jsonPathRegexp, "regexp for extracting _path from json log name (when -inferpath=true)")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")

	// Flags added for readers are -i XXX json
	c.ReaderFlags.SetFlags(f)

	return c, nil
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

//XXX lots here copied from zq command... we should refactor into a tools package
func (c *Command) Run(args []string) error {
	//XXX
	if c.outputFile == "-" {
		c.outputFile = ""
	}

	ark, err := archive.OpenArchive(c.root)
	if err != nil {
		return err
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
	if len(args) == 0 {
		return errors.New("zar zq needs input arguments")
	}
	// XXX this is parallelizable except for writing to stdout when
	// concatenating results
	return archive.Walk(ark, func(zardir string) error {
		inputs := args
		var query ast.Proc
		var err error
		first := archive.Localize(zardir, inputs[:1])
		if first[0] != "" && fileExists(first[0]) {
			query, err = zql.ParseProc("*")
			if err != nil {
				return err
			}
		} else {
			query, err = zql.ParseProc(inputs[0])
			if err != nil {
				return err
			}
			inputs = inputs[1:]
		}
		localPaths := archive.Localize(zardir, inputs)
		readers, err := c.inputReaders(resolver.NewContext(), localPaths)
		if err != nil {
			return err
		}
		if len(readers) == 0 {
			// skip and warn if no inputs found
			if !c.quiet {
				fmt.Fprintf(os.Stderr, "%s: no inputs files found\n", zardir)
			}
			return nil
		}
		wch := make(chan string, 5)
		if !c.stopErr {
			for i, r := range readers {
				readers[i] = zbuf.NewWarningReader(r, wch)
			}
		}
		reader := zbuf.NewCombiner(readers)
		defer reader.Close()
		writer, err := c.openOutput(zardir, c.outputFile)
		if err != nil {
			return err
		}
		defer writer.Close()
		// XXX we shouldn't need zap here, nano?  etc
		mux, err := driver.CompileWarningsCh(context.Background(), query, reader, false, nano.MaxSpan, zap.NewNop(), wch)
		if err != nil {
			return err
		}
		d := driver.NewCLI(writer)
		if !c.quiet {
			d.SetWarningsWriter(os.Stderr)
		}
		return driver.Run(mux, d, nil)
	})
}

func (c *Command) inputReaders(zctx *resolver.Context, paths []string) ([]zbuf.Reader, error) {
	cfg := detector.OpenConfig{
		Format:         c.ReaderFlags.Format,
		JSONTypeConfig: c.jsonTypeConfig,
		JSONPathRegex:  c.jsonPathRegexp,
	}
	var readers []zbuf.Reader
	for _, path := range paths {
		file, err := detector.OpenFile(zctx, path, cfg)
		if err != nil {
			if os.IsNotExist(err) {
				if !c.quiet {
					fmt.Fprintf(os.Stderr, "warning: %s not found\n", path)
				}
				continue
			}
			err = fmt.Errorf("%s: %w", path, err)
			if c.stopErr {
				return nil, err
			}
			fmt.Fprintf(os.Stderr, "%s\n", err)
			continue
		}
		// wrap in a named reader so the reader implements Stringer,
		// e.g., as used by scanner.Combiner
		readers = append(readers, namedReader{file, path})
	}
	return readers, nil
}

func (c *Command) openOutput(zardir, filename string) (zbuf.WriteCloser, error) {
	path := filename
	// prepend path if not stdout
	if path != "" {
		path = filepath.Join(zardir, filename)
	}
	flags := zio.WriterFlags{Format: "zng"}
	w, err := emitter.NewFile(path, &flags)
	if err != nil {
		return nil, err
	}
	return w, nil
}

type namedReader struct {
	zbuf.Reader
	name string
}

func (r namedReader) String() string {
	return r.name
}
