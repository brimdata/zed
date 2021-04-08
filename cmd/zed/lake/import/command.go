package zarimport

import (
	"errors"
	"flag"
	"os"

	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/procflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/signalctx"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
)

var Import = &charm.Spec{
	Name:  "import",
	Usage: "import [-R root] [options] [file|S3-object|- ...]",
	Short: "import log files into pieces",
	Long: `
The import command provides a way to create a new zar archive with ZNG data
from an existing file, S3 location, or stdin.

The input data is sorted and partitioned by time into approximately equal
sized ZNG files, called "chunks". The path of each chunk is a subdirectory in
the specified root location (-R or ZED_LAKE_ROOT), where the subdirectory name is
derived from the timestamp of the first zng record in that chunk.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Import)
}

type Command struct {
	*zedlake.Command
	asc                   bool
	root                  string
	dataPath              string
	thresh                units.Bytes
	importBufSize         units.Bytes
	importStreamRecordMax int
	empty                 bool
	inputFlags            inputflags.Flags
	procFlags             procflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	f.BoolVar(&c.asc, "asc", false, "store archive data in ascending order")
	f.StringVar(&c.root, "R", os.Getenv("ZED_LAKE_ROOT"), "root location of zar archive to walk")
	f.StringVar(&c.dataPath, "data", "", "location for storing data files (defaults to root)")
	c.thresh = lake.DefaultLogSizeThreshold
	f.Var(&c.thresh, "s", "target size of chunk files, as '10MB' or '4GiB', etc.")
	c.importBufSize = units.Bytes(lake.ImportBufSize)
	f.Var(&c.importBufSize, "bufsize", "maximum size of data read into memory before flushing to disk, as '99MB', '4GiB', etc.")
	f.IntVar(&c.importStreamRecordMax, "streammax", lake.ImportStreamRecordsMax, "limit for number of records in each ZNG stream (0 for no limit)")
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
	} else if !c.empty && len(args) == 0 {
		return errors.New("zar import: at least one input file must be specified (- for stdin)")
	}
	lake.ImportBufSize = int64(c.importBufSize)
	lake.ImportStreamRecordsMax = c.importStreamRecordMax

	thresh := int64(c.thresh)
	co := &lake.CreateOptions{
		DataPath:         c.dataPath,
		SortAscending:    c.asc,
		LogSizeThreshold: &thresh,
	}

	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}

	lk, err := lake.CreateOrOpenLake(c.root, co, nil)
	if err != nil {
		return err
	}

	if c.empty {
		return nil
	}

	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()

	paths := args
	zctx := zson.NewContext()
	readers, err := c.inputFlags.Open(zctx, paths, false)
	if err != nil {
		return err
	}
	defer zbuf.CloseReaders(readers)
	reader, err := zbuf.MergeReadersByTsAsReader(ctx, readers, lk.DataOrder)
	if err != nil {
		return err
	}
	return lake.Import(ctx, lk, zctx, reader)
}
