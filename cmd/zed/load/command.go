package load

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/display"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/zio"
	"github.com/paulbellamy/ratecounter"
	"golang.org/x/term"
)

var Cmd = &charm.Spec{
	Name:  "load",
	Usage: "load [options] file|S3-object|- ...",
	Short: "add and commit data to a branch",
	Long: `
The load command adds data to a pool and commits it to a branch.
`,
	New: New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	commit bool
	cli.CommitFlags
	procFlags  procflags.Flags
	inputFlags inputflags.Flags
	lakeFlags  lakeflags.Flags

	// status output
	ctx       context.Context
	rate      *ratecounter.RateCounter
	statsers  []*inputflags.StatsReader
	totalRead int64
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*root.Command),
	}
	c.CommitFlags.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	c.inputFlags.SetFlags(f, true)
	c.procFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.inputFlags, &c.procFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return errors.New("zed load: at least one input file must be specified (- for stdin)")
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	paths := args
	local := storage.NewLocalEngine()
	c.statsers, err = c.inputFlags.OpenWithStats(ctx, zed.NewContext(), local, paths, false)
	if err != nil {
		return err
	}
	readers := make([]zio.Reader, len(c.statsers))
	for i, r := range c.statsers {
		readers[i] = r
	}
	defer zio.CloseReaders(readers)
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	if head.Pool == "" {
		return lakeflags.ErrNoHEAD
	}
	poolID, err := lake.PoolID(ctx, head.Pool)
	if err != nil {
		return err
	}
	var d *display.Display
	if !c.lakeFlags.Quiet && term.IsTerminal(int(os.Stderr.Fd())) {
		c.ctx = ctx
		c.rate = ratecounter.NewRateCounter(time.Second)
		d = display.New(c, time.Second/2, os.Stderr)
		go d.Run()
	}
	commitID, err := lake.Load(ctx, poolID, head.Branch, zio.ConcatReader(readers...), c.CommitMessage())
	if d != nil {
		d.Close()
	}
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s committed\n", commitID)
	}
	return nil
}

type displayer struct {
	statsers  []*inputflags.StatsReader
	totalRead int64
	rate      *ratecounter.RateCounter
	ctx       context.Context
}

// (1/1) 1GB/4GB 87%

func (c *Command) Display(w io.Writer) bool {
	var completed int
	var totalBytes, readBytes units.Bytes
	for _, statser := range c.statsers {
		total, read := statser.BytesTotal, statser.BytesRead()
		if total == read {
			completed++
		}
		totalBytes += units.Bytes(total)
		readBytes += units.Bytes(read)
	}
	fmt.Fprintf(w, "(%d/%d) ", completed, len(c.statsers))
	rate := c.incrRate(readBytes)
	if totalBytes == 0 {
		fmt.Fprintf(w, "%s %s/s\n", readBytes.Abbrev(), rate.Abbrev())
	} else {
		fmt.Fprintf(w, "%s/%s %s/s %.2f%%\n", readBytes.Abbrev(), totalBytes.Abbrev(), rate.Abbrev(), float64(readBytes)/float64(totalBytes)*100)
	}
	return c.ctx.Err() == nil

}

func (c *Command) incrRate(readBytes units.Bytes) units.Bytes {
	c.rate.Incr(int64(readBytes) - c.totalRead)
	c.totalRead = int64(readBytes)
	return units.Bytes(c.rate.Rate())
}
