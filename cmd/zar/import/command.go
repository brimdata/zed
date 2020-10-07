package zarimport

import (
	"errors"
	"flag"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cli/inputflags"
	"github.com/brimsec/zq/cli/procflags"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/brimsec/zq/pkg/rlimit"
	"github.com/brimsec/zq/pkg/signalctx"
	"github.com/brimsec/zq/pkg/units"
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
	root          string
	dataPath      string
	thresh        units.Bytes
	importBufSize units.Bytes
	empty         bool
	inputFlags    inputflags.Flags
	procFlags     procflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root location of zar archive to walk")
	f.StringVar(&c.dataPath, "data", "", "location for storing data files (defaults to root)")
	c.thresh = archive.DefaultLogSizeThreshold
	f.Var(&c.thresh, "s", "target size of chunk files, as '10MB' or '4GiB', etc.")
	c.importBufSize = units.Bytes(archive.ImportBufSize)
	f.Var(&c.importBufSize, "bufsize", "maximum size of data read into memory before flushing to disk '99MB', '4GiB', etc.")
	f.BoolVar(&c.empty, "empty", false, "create an archive without initial data")
	c.inputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.inputFlags, &c.procFlags); err != nil {
		return err
	}
	if c.empty && len(args) > 0 {
		return errors.New("zar import: empty flag specified with input files")
	} else if !c.empty && len(args) != 1 {
		return errors.New("zar import: exactly one input file must be specified (- for stdin)")
	}
	archive.ImportBufSize = int64(c.importBufSize)

	co := &archive.CreateOptions{DataPath: c.dataPath}
	thresh := int64(c.thresh)
	co.LogSizeThreshold = &thresh

	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
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
	reader, err := detector.OpenFile(zctx, path, c.inputFlags.Options())
	if err != nil {
		return err
	}
	defer reader.Close()

	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()

	return archive.Import(ctx, ark, zctx, reader)
}
