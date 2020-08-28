package zarimport

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/alecthomas/units"
	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/pkg/signalctx"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Import = &charm.Spec{
	Name:  "import",
	Usage: "import [-R root] [options] [file|S3-object|-]",
	Short: "import log files into pieces",
	Long: `
The import command provides a way to create a new zar archive with ZNG data
from an existing file, S3 location, or stdin.

The input data is sorted and partitioned by time into approximately equal
sized ZNG files, called "chunks". The path of each chunk is a subdirectory in
the specified root location (-R or ZAR_ROOT), where the subdirectory name is
derived from the timestamp of the first zng record in that chunk.
`,
	New: New,
}

func init() {
	root.Zar.Add(Import)
}

type Command struct {
	*root.Command
	root            string
	dataPath        string
	sortMemMaxBytes int
	thresh          string
	verbose         bool
	empty           bool
	ReaderFlags     zio.ReaderFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.BoolVar(&c.verbose, "v", false, "show verbose details")
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.StringVar(&c.dataPath, "data", "", "location for storing data files (defaults to root)")
	f.IntVar(&c.sortMemMaxBytes, "sortmem", sort.MemMaxBytes, "maximum memory used by sort, in bytes")
	f.StringVar(&c.thresh, "s", units.Base2Bytes(archive.DefaultLogSizeThreshold).String(), "target size of chunk files, as '10MB' or '4GiB', etc.")
	f.BoolVar(&c.empty, "empty", false, "create an archive without initial data")
	c.ReaderFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	if c.empty && len(args) > 0 {
		return errors.New("zar import: empty flag specified with input files")
	} else if !c.empty && len(args) != 1 {
		return errors.New("zar import: exactly one input file must be specified (- for stdin)")
	}

	if c.sortMemMaxBytes <= 0 {
		return errors.New("sortmem value must be greater than zero")
	}
	sort.MemMaxBytes = c.sortMemMaxBytes

	co := &archive.CreateOptions{DataPath: c.dataPath}
	if thresh, err := units.ParseStrictBytes(c.thresh); err != nil {
		return fmt.Errorf("invalid target file size: %w", err)
	} else {
		co.LogSizeThreshold = &thresh
	}

	ark, err := archive.CreateOrOpenArchive(c.root, co, nil)
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
	defer reader.Close()

	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()

	return archive.Import(ctx, ark, zctx, reader)
}
