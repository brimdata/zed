package cmdzdx

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Zdx = &charm.Spec{
	Name:  "zdx",
	Usage: "zdx [-R dir] [options] file",
	Short: "walk an archive and create zdx indexes for the named file",
	Long: `
"zar zdx" descends the directory given by the -R option (or ZAR_ROOT env) looking for
logs with zar directories and for each such directory found, it runs
zdx on the file provided relative to each zar directory.
The input file must have a field called "key" where all the records in the
file are sorted by that key in increasing value according to the zng type
of the key.
If the root directory is not specified by either the ZAR_ROOT environemnt
variable or the -R option, then the current directory is assumed.
`,
	New: New,
}

func init() {
	root.Zar.Add(Zdx)
}

type Command struct {
	*root.Command
	root        string
	framesize   int
	outputFile  string
	quiet       bool
	ReaderFlags zio.ReaderFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in zdx file")
	f.StringVar(&c.outputFile, "o", "zdx", "output zdx bundle name")
	f.BoolVar(&c.quiet, "q", false, "do not print any warnings to stderr")

	c.ReaderFlags.SetFlags(f)

	return c, nil
}

//XXX lots here copied from zq command... we should refactor into a tools package
func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar zdx takes exactly one input file name")
	}
	rootPath := c.root
	if rootPath == "" {
		rootPath = "."
	}
	// XXX this is parallelizable except for writing to stdout when
	// concatenating results
	return archive.Walk(rootPath, func(zardir string) error {
		path := archive.Localize(zardir, args[:1])
		zctx := resolver.NewContext()
		cfg := detector.OpenConfig{
			Format:    c.ReaderFlags.Format,
			DashStdin: true, //XXX
			//JSONTypeConfig: c.jsonTypeConfig,
			//JSONPathRegex:  c.jsonPathRegexp,
		}
		file, err := detector.OpenFile(zctx, path[0], cfg)
		if err != nil {
			if os.IsNotExist(err) {
				if !c.quiet {
					fmt.Fprintln(os.Stderr, err.Error())
				}
				err = nil
			}
			return err
		}

		//XXX should have a single-file Localizer
		outputPath := archive.Localize(zardir, []string{c.outputFile})
		writer, err := zdx.NewWriter(outputPath[0], c.framesize)
		if err != nil {
			return err
		}
		close := true
		defer func() {
			if close {
				writer.Close()
			}
		}()
		reader := zbuf.Reader(file)
		// XXX we can add this later... for now, "zar index" handles
		// this is the code path here demos the creation of indexes
		// with abritray aggrates in the fields of each base record
		//if !c.inputReady {
		//	reader, err = c.buildTable(zctx, file)
		//	if err != nil {
		//		return err
		//	}
		//}
		if err := zbuf.Copy(writer, reader); err != nil {
			return err
		}
		close = false
		return writer.Close()
	})
}
