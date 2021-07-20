package add

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/lakeflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var Add = &charm.Spec{
	Name:  "add",
	Usage: "add [options] path [path ...]",
	Short: "add data to a pool",
	Long: `
The add command adds data to a pool from an existing file, S3 location, or stdin.

One or more data sources may be specified by path.
The path may be a file on the local file system, an S3 URI,
or "-" for standard input.  Standard input may be mixed with
other path inputs.

By default, data is deposited into the pool's staging area the
a pending "commit tag" is displayed.  This data can then be commited
to the lake automically with the "zed lake commit" command.

If the "-commit" flag is given, then the data is commited to the lake atomically after
all data has been sucessfully written.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Add)
	zedapi.Cmd.Add(Add)
}

// TBD: add option to apply Zed program on add path?

type Command struct {
	lake       zedlake.Command
	seekStride units.Bytes
	commit     bool
	inputFlags inputflags.Flags
	lakeFlags  lakeflags.Flags
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		lake:       parent.(zedlake.Command),
		seekStride: units.Bytes(lake.SeekIndexStride),
	}
	f.BoolVar(&c.commit, "commit", false, "commit added data if successfully written")
	f.Var(&c.seekStride, "seekstride", "size of seek-index unit for ZNG data, as '32KB', '1MB', etc.")
	c.inputFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init(&c.inputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("zed lake add: at least one input file must be specified (- for stdin)")
	}
	lake.SeekIndexStride = int(c.seekStride)
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	paths := args
	local := storage.NewLocalEngine()
	readers, err := c.inputFlags.Open(zson.NewContext(), local, paths, false)
	if err != nil {
		return err
	}
	defer zio.CloseReaders(readers)
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	pool, err := lake.LookupPool(ctx, c.lakeFlags.PoolName)
	// XXX See issue #2921.  Not clear we should merge by ts here.
	reader, err := zbuf.MergeReadersByTsAsReader(ctx, readers, pool.Layout.Order)
	if err != nil {
		return err
	}
	action := "staged"
	var commit *api.CommitRequest
	if c.commit {
		commit = c.CommitRequest()
		action = "committed"
	}
	id, err := lake.Add(ctx, pool.ID, reader, commit)
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s %s\n", id, action)
	}
	return nil
}
