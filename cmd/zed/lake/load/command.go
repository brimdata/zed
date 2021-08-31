package load

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/procflags"
	zedapi "github.com/brimdata/zed/cmd/zed/api"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var Load = &charm.Spec{
	Name:  "load",
	Usage: "load [options] -p pool[@branch] file|S3-object|- ...",
	Short: "add and commit data to a pool",
	Long: `
The load command adds data to a pool and commits it in one automatic operation.
See documentation on "zed lake add" and "zed lake commit" for more details.

A pool and branch must be specified with the -p option.  If a pool name is given
without a branch name, then the branch is assumed to be "main".
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Load)
	zedapi.Cmd.Add(Load)
}

type Command struct {
	lake       zedlake.Command
	seekStride units.Bytes
	commit     bool
	zedlake.CommitFlags
	procFlags  procflags.Flags
	inputFlags inputflags.Flags
	lakeFlags  lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		lake:       parent.(zedlake.Command),
		seekStride: units.Bytes(lake.SeekIndexStride),
	}
	f.Var(&c.seekStride, "seekstride", "size of seek-index unit for ZNG data, as '32KB', '1MB', etc.")
	c.CommitFlags.SetFlags(f)
	c.inputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Root().Init(&c.inputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("zed lake load: at least one input file must be specified (- for stdin)")
	}
	lake.SeekIndexStride = int(c.seekStride)
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.lake.Open(ctx)
	if err != nil {
		return err
	}
	paths := args
	local := storage.NewLocalEngine()
	readers, err := c.inputFlags.Open(zson.NewContext(), local, paths, false)
	if err != nil {
		return err
	}
	defer zio.CloseReaders(readers)
	poolName, branchName := c.lakeFlags.Branch()
	if poolName == "" {
		return errors.New("pool name must be specified with -p")
	}
	poolID, err := lake.PoolID(ctx, poolName)
	if err != nil {
		return err
	}
	commitID, err := lake.Load(ctx, poolID, branchName, zio.ConcatReader(readers...), c.CommitMessage())
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s committed\n", commitID)
	}
	return nil
}
