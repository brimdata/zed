package add

import (
	"errors"
	"flag"
	"fmt"
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

var Load = &charm.Spec{
	Name:  "load",
	Usage: "load [-R root] [-p pool] [options] file|S3-object|- ...",
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
	*zedlake.Command
	importBufSize         units.Bytes
	importStreamRecordMax int
	commit                bool
	lakeFlags             zedlake.Flags
	zedlake.CommitFlags
	procFlags  procflags.Flags
	inputFlags inputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zedlake.Command)}
	c.importBufSize = units.Bytes(lake.ImportBufSize)
	f.Var(&c.importBufSize, "bufsize", "maximum size of data read into memory before flushing to disk, as '99MB', '4GiB', etc.")
	f.IntVar(&c.importStreamRecordMax, "streammax", lake.ImportStreamRecordsMax, "limit for number of records in each ZNG stream (0 for no limit)")
	c.lakeFlags.SetFlags(f)
	c.CommitFlags.SetFlags(f)
	c.inputFlags.SetFlags(f)
	c.procFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.inputFlags, &c.procFlags); err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("zed lake load: at least one input file must be specified (- for stdin)")
	}
	lake.ImportBufSize = int64(c.importBufSize)
	lake.ImportStreamRecordsMax = c.importStreamRecordMax
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	ctx, cancel := signalctx.New(os.Interrupt)
	defer cancel()
	pool, err := c.lakeFlags.OpenPool(ctx)
	if err != nil {
		return err
	}
	paths := args
	zctx := zson.NewContext()
	readers, err := c.inputFlags.Open(zctx, paths, false)
	if err != nil {
		return err
	}
	defer zbuf.CloseReaders(readers)
	reader, err := zbuf.MergeReadersByTsAsReader(ctx, readers, pool.Order)
	if err != nil {
		return err
	}
	commitID, err := pool.Add(ctx, zctx, reader)
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
	if !c.lakeFlags.Quiet {
		fmt.Println("commit successful", commitID)
		for _, action := range txn {
			//XXX clean this up and allow -f output
			fmt.Printf("  %s\n", action)
		}
	}
	return nil
}
