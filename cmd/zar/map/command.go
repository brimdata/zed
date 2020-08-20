package zarmap

import (
	"errors"
	"flag"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/signalctx"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/mccanne/charm"
)

var Map = &charm.Spec{
	Name:  "map",
	Usage: "map [-R root] [options] [zql] file [file...]",
	Short: "execute ZQL for each archive directory",
	Long: `
"zar map" executes a ZQL query on one or more files in each of the
chunk directories of a zar archive, sending its output to either stdout,
or to a per-directory file, specified via "-o". Input file names are
relative to each zar subdirectory, and the special name "_" refers to
the chunk file itself.
`,
	New: New,
}

func init() {
	root.Zar.Add(Map)
}

type Command struct {
	*root.Command
	forceBinary  bool
	outputFile   string
	quiet        bool
	root         string
	stopErr      bool
	textShortcut bool
	writerFlags  zio.WriterFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.BoolVar(&c.forceBinary, "B", false, "allow binary zng be sent to a terminal output")
	f.StringVar(&c.outputFile, "o", "", "output file relative to zar directory")
	f.BoolVar(&c.quiet, "q", false, "don't display zql warnings")
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.BoolVar(&c.stopErr, "e", true, "stop upon input errors")
	f.BoolVar(&c.textShortcut, "t", false, "use format tzng independent of -f option")
	c.writerFlags.SetFlags(f)
	return c, nil
}

//XXX lots here copied from zq command... we should refactor into a tools package
func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("zar map needs input arguments")
	}

	if c.outputFile == "-" {
		c.outputFile = ""
	}
	if c.textShortcut {
		c.writerFlags.Format = "tzng"
	}
	if c.outputFile == "" && c.writerFlags.Format == "zng" && emitter.IsTerminal(os.Stdout) && !c.forceBinary {
		return errors.New("map: writing binary zng data to terminal; override with -B or use -t for text.")
	}

	// Don't allow non-zng to be written inside the archive.
	if c.outputFile != "" && c.writerFlags.Format != "zng" {
		return errors.New("zq: only ZNG format allowed for chunk associated files")
	}

	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()

	ark, err := archive.OpenArchive(c.root, nil)
	if err != nil {
		return err
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
		writer, err := c.openOutput(zardir)
		if err != nil {
			return err
		}
		defer writer.Close()
		d := driver.NewCLI(writer)
		if !c.quiet {
			d.SetWarningsWriter(os.Stderr)
		}
		return driver.Run(ctx, d, query, zctx, reader, driver.Config{
			ReaderSortKey:     "ts",
			ReaderSortReverse: ark.DataSortDirection == zbuf.DirTimeReverse,
			Warnings:          wch,
		})
	})
}

func (c *Command) openOutput(zardir iosrc.URI) (zbuf.WriteCloser, error) {
	path := ""
	if c.outputFile != "" {
		path = zardir.AppendPath(c.outputFile).String()
	}
	w, err := emitter.NewFile(path, &c.writerFlags)
	if err != nil {
		return nil, err
	}
	return w, nil
}
