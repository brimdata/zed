package ls

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/brimdata/zed/cli/outputflags"
	zedlake "github.com/brimdata/zed/cmd/zed/lake"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

var Ls = &charm.Spec{
	Name:  "ls",
	Usage: "ls [options]",
	Short: "list objects in data pool",
	Long: `
"zed lake ls" shows a listing of a data pool's segments as tags.
The the pool flag "-p" is not given, then the lake's pools are listed.
`,
	New: New,
}

func init() {
	zedlake.Cmd.Add(Ls)
}

type Command struct {
	lake        *zedlake.Command
	partition   bool
	at          string
	outputFlags outputflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{lake: parent.(*zedlake.Command)}
	f.StringVar(&c.at, "at", "", "commit tag or journal ID for time travel")
	f.BoolVar(&c.partition, "partition", false, "display partitions as determined by scan logic")
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) > 0 {
		return errors.New("zed lake ls: too many arguments")
	}
	ctx, cleanup, err := c.lake.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	pipeReader, pipeWriter := io.Pipe()
	w := zngio.NewWriter(pipeWriter, zngio.WriterOpts{})
	local := storage.NewLocalEngine()
	if c.lake.Flags.PoolName == "" {
		lk, err := c.lake.Flags.Open(ctx, local)
		if err != nil {
			return err
		}
		go func() {
			lk.ScanPools(ctx, w)
			w.Close()
		}()
	} else {
		pool, err := c.lake.Flags.OpenPool(ctx, local)
		if err != nil {
			return err
		}
		var at journal.ID
		if c.at != "" {
			at, err = zedlake.ParseJournalID(ctx, pool, c.at)
			if err != nil {
				return fmt.Errorf("zed lake ls: %w", err)
			}
		}
		snap, err := pool.Log().Snapshot(ctx, at)
		if err != nil {
			return err
		}
		if c.partition {
			go func() {
				pool.ScanPartitions(ctx, w, snap, nil)
				w.Close()
			}()
		} else {
			go func() {
				pool.ScanSegments(ctx, w, snap, nil)
				w.Close()
			}()
		}
	}
	r := zngio.NewReader(pipeReader, zson.NewContext())
	return zedlake.CopyToOutput(ctx, local, c.outputFlags, r)
}
