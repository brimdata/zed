package add

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed/cli/inputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
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
}

// TBD: add option to apply Zed program on add path?

type Command struct {
	lake                  *zedlake.Command
	importStreamRecordMax int
	commit                bool
	inputFlags            inputflags.Flags
	zedlake.CommitFlags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.BoolVar(&c.commit, "commit", false, "commit added data if successfully written")
	f.IntVar(&c.importStreamRecordMax, "streammax", lake.ImportStreamRecordsMax, "limit for number of records in each ZNG stream (0 for no limit)")
	c.inputFlags.SetFlags(f)
	c.CommitFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.lake.Init(&c.inputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("zed lake add: at least one input file must be specified (- for stdin)")
	}
	lake.ImportStreamRecordsMax = c.importStreamRecordMax
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	pool, err := c.lake.Flags.OpenPool(ctx)
	if err != nil {
		return err
	}
	paths := args
	readers, err := c.inputFlags.Open(zson.NewContext(), paths, false)
	if err != nil {
		return err
	}
	defer zio.CloseReaders(readers)
	reader, err := zbuf.MergeReadersByTsAsReader(ctx, readers, pool.Order)
	if err != nil {
		return err
	}
	commit, err := pool.Add(ctx, reader)
	if err != nil {
		return err
	}
	if c.commit {
		if err := pool.Commit(ctx, commit, c.Date.Ts(), c.User, c.Message); err != nil {
			return err
		}
		if !c.lake.Flags.Quiet {
			fmt.Printf("%s committed\n", commit)
		}
		return nil
	}
	if !c.lake.Flags.Quiet {
		fmt.Printf("%s staged\n", commit)
	}
	return nil
}
