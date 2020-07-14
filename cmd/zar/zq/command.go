package zq

import (
	"context"
	"errors"
	"flag"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
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
	root       string
	outputFile string
	stopErr    bool
	quiet      bool
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.BoolVar(&c.quiet, "q", false, "don't display zql warnings")
	f.StringVar(&c.outputFile, "o", "", "write data to output file")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")

	return c, nil
}

//XXX lots here copied from zq command... we should refactor into a tools package
func (c *Command) Run(args []string) error {
	//XXX
	if c.outputFile == "-" {
		c.outputFile = ""
	}

	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("zar zq needs input arguments")
	}
	// XXX this is parallelizable except for writing to stdout when
	// concatenating results
	return archive.Walk(ark, func(zardir iosrc.URI) error {
		inputs := args
		var query ast.Proc
		first := archive.Localize(zardir, inputs[0])
		ok, err := iosrc.Exists(first)
		if err != nil {
			return err
		}
		if ok {
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
		var paths []string
		for _, input := range inputs {
			p := archive.Localize(zardir, input)
			// XXX Doing this because detector doesn't support file uri's. At
			// some point it should.
			if p.Scheme == "file" {
				paths = append(paths, p.Filepath())
			} else {
				paths = append(paths, p.String())
			}
		}
		zctx := resolver.NewContext()
		cfg := detector.OpenConfig{Format: "zng"}
		rc := detector.MultiFileReader(zctx, paths, cfg)
		defer rc.Close()
		reader := zbuf.Reader(rc)
		wch := make(chan string, 5)
		if !c.stopErr {
			reader = zbuf.NewWarningReader(reader, wch)
		}
		writer, err := c.openOutput(zardir, c.outputFile)
		if err != nil {
			return err
		}
		defer writer.Close()
		mux, err := driver.Compile(context.Background(), query, reader, driver.Config{
			TypeContext:       zctx,
			ReaderSortKey:     "ts",
			ReaderSortReverse: ark.DataSortDirection == zbuf.DirTimeReverse,
			Warnings:          wch,
		})
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

func (c *Command) openOutput(zardir iosrc.URI, filename string) (zbuf.WriteCloser, error) {
	path := filename
	// prepend path if not stdout
	if path != "" {
		path = zardir.AppendPath(filename).String()
	}
	w, err := emitter.NewFile(path, &zio.WriterFlags{Format: "zng"})
	if err != nil {
		return nil, err
	}
	return w, nil
}
