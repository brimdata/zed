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
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
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
	root        string
	dataPath    string
	thresh      string
	empty       bool
	ReaderFlags zio.ReaderFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.root, "R", os.Getenv("ZAR_ROOT"), "root directory of zar archive to walk")
	f.StringVar(&c.dataPath, "data", "", "location for storing data files (defaults to root directory)")
	f.StringVar(&c.thresh, "s", units.Base2Bytes(archive.DefaultLogSizeThreshold).String(), "target size of chopped files, as '10MB' or '4GiB', etc.")
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
