package add

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/procflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var Load = &charm.Spec{
	Name:  "load",
	Usage: "load [options] file|S3-object|- ...",
	Short: "add and commit data to a pool",
	Long: `
The load command adds data to a pool and commits it in one automatic operation.
See documentation on "zed lake add" and "zed lake commit" for more details.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Load)
}

type Command struct {
	lake                  *zedlake.Command
	importStreamRecordMax int
	commit                bool
	zedlake.CommitFlags
	procFlags  procflags.Flags
	inputFlags inputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.IntVar(&c.importStreamRecordMax, "streammax", lake.ImportStreamRecordsMax, "limit for number of records in each ZNG stream (0 for no limit)")
	c.CommitFlags.SetFlags(f)
	c.inputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.inputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("zed lake load: at least one input file must be specified (- for stdin)")
	}
	lake.ImportStreamRecordsMax = c.importStreamRecordMax
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	local := storage.NewLocalEngine()
	pool, err := c.lake.Flags.OpenPool(ctx, local)
	if err != nil {
		return err
	}
	paths := args
	readers, err := c.inputFlags.Open(zson.NewContext(), local, paths, false)
	if err != nil {
		return err
	}
	defer zio.CloseReaders(readers)
	reader, err := zbuf.MergeReadersByTsAsReader(ctx, readers, pool.Layout.Order)
	if err != nil {
		return err
	}
	commitID, err := pool.Add(ctx, reader)
	if err != nil {
		return err
	}
	txn, err := pool.LoadFromStaging(ctx, commitID)
	if err != nil {
		return err
	}
	if err := pool.Commit(ctx, commitID, c.Date.Ts(), c.User, c.Message); err != nil {
		return err
	}
	if !c.lake.Flags.Quiet {
		fmt.Printf("%s committed %d segments\n", commitID, len(txn.Actions))
	}
	return nil
}
