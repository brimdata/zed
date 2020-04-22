package index

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Chop = &charm.Spec{
	Name:  "chop",
	Usage: "chop [options] file",
	Short: "chop log files into pieces",
	Long: `
The chop command provides a crude way to break up a zng file or stream
into smaller chunks.  It takes as input zng data and cuts the stream
into chunks where each chunk is created when the size threshold is exceeded,
either in bytes (-b) or megabytes (-s).  The path of each chunk is a subdirectory
in the specified directory (-d) where the subdirectory name is derived from the
timestamp of the first zng record in that chunk.
`,
	New: New,
}

func init() {
	root.Zar.Add(Chop)
}

type Command struct {
	*root.Command
	megaThresh  int
	byteThresh  int
	dir         string
	quiet       bool
	ReaderFlags zio.ReaderFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.dir, "d", ".", "destination directory for chopped files")
	f.IntVar(&c.megaThresh, "s", 500, "target size of chopped files in MB")
	f.IntVar(&c.byteThresh, "b", 0, "target size of chopped files in bytes (overrides -s)")
	f.BoolVar(&c.quiet, "q", false, "do not print progress updates to stdout")
	c.ReaderFlags.SetFlags(f)
	return c, nil
}

func tsDir(ts nano.Ts) string {
	return ts.Time().Format("20060102")
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar chop: exactly one input file must be specified (- for stdin)")
	}
	path := args[0]
	zctx := resolver.NewContext()
	cfg := detector.OpenConfig{
		Format:    c.ReaderFlags.Format,
		DashStdin: true,
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
	thresh := c.byteThresh
	if thresh == 0 {
		thresh = c.megaThresh * 1024 * 1024
	}
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
			dir := filepath.Join(c.dir, tsDir(ts))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			path := filepath.Join(dir, ts.StringFloat()+".zng")
			//XXX for now just truncate any existing file.
			// a future PR will do a split/merge.
			out, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
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
		if n >= thresh {
			if err := w.Close(); err != nil {
				return err
			}
			w = nil
			n = 0
		}
	}
}
