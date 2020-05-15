package zarimport

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Import = &charm.Spec{
	Name:  "import",
	Usage: "import [options] file",
	Short: "import log files into pieces",
	Long: `
The import command provides a way to create a new zar archive with data.
It takes as input zng data and cuts the stream
into chunks where each chunk is created when the size threshold is exceeded,
either in bytes (-b) or megabytes (-s).  The path of each chunk is a subdirectory
in the specified directory (-R or ZAR_ROOT) where the subdirectory name is derived from the
timestamp of the first zng record in that chunk.
`,
	New: New,
}

func init() {
	root.Zar.Add(Import)
}

type Command struct {
	*root.Command
	megaThresh  int
	byteThresh  int
	root        string
	quiet       bool
	empty       bool
	ReaderFlags zio.ReaderFlags
	CreateFlags archive.CreateFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive for chopped files")
	f.BoolVar(&c.quiet, "q", false, "do not print progress updates to stdout")
	f.BoolVar(&c.empty, "empty", false, "create an archive without initial data")
	c.ReaderFlags.SetFlags(f)
	c.CreateFlags.SetFlags(f)
	return c, nil
}

func tsDir(ts nano.Ts) string {
	return ts.Time().Format("20060102")
}

func (c *Command) Run(args []string) error {
	if c.empty && len(args) > 0 {
		return errors.New("zar import: empty flag specified with input files")
	} else if !c.empty && len(args) != 1 {
		return errors.New("zar import: exactly one input file must be specified (- for stdin)")
	}

	ark, err := archive.CreateOrOpenArchive(c.root, c.CreateFlags)
	if err != nil {
		return err
	}

	if c.empty {
		return nil
	}

	path := args[0]
	if path == "-" {
		path = detector.StdinPath
	}
	zctx := resolver.NewContext()
	cfg := detector.OpenConfig{
		Format: c.ReaderFlags.Format,
		//JSONTypeConfig: c.jsonTypeConfig,
		//JSONPathRegex:  c.jsonPathRegexp,
	}
	reader, err := detector.OpenFile(zctx, path, cfg)
	if err != nil {
		return err
	}
	var w *bufwriter.Writer
	var zw zbuf.Writer
	var n int
	for {
		rec, err := reader.Read()
		if err != nil || rec == nil {
			if w != nil {
				if err := w.Close(); err != nil {
					return err
				}
			}
			return err
		}
		if w == nil {
			ts := rec.Ts
			dir := filepath.Join(ark.Root, tsDir(ts))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			path := filepath.Join(dir, ts.StringFloat()+".zng")
			//XXX for now just truncate any existing file.
			// a future PR will do a split/merge.
			out, err := fs.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return err
			}
			if !c.quiet {
				fmt.Printf("writing %s\n", path)
			}
			w = bufwriter.New(out)
			zw = zngio.NewWriter(w, zio.WriterFlags{})
		}
		if err := zw.Write(rec); err != nil {
			return err
		}
		n += len(rec.Raw)
		if n >= ark.Config.LogSizeThreshold {
			if err := w.Close(); err != nil {
				return err
			}
			w = nil
			n = 0
		}
	}
}
