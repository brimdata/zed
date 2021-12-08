package load

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/procflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/zio"
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
	seekStride units.Bytes
	commit     bool
	cli.CommitFlags
	procFlags  procflags.Flags
	inputFlags inputflags.Flags
	lakeFlags  lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command:    parent.(*root.Command),
		seekStride: units.Bytes(lake.SeekIndexStride),
	}
	f.Var(&c.seekStride, "seekstride", "size of seek-index unit for ZNG data, as '32KB', '1MB', etc.")
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
		return errors.New("zed lake load: at least one input file must be specified (- for stdin)")
	}
	lake.SeekIndexStride = int(c.seekStride)
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return err
	}
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	paths := args
	local := storage.NewLocalEngine()
	readers, err := c.inputFlags.Open(ctx, zed.NewContext(), local, paths, false)
	if err != nil {
		return err
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
	commitID, err := lake.Load(ctx, poolID, head.Branch, zio.ConcatReader(readers...), c.CommitMessage())
	if err != nil {
		return err
	}
	if !c.lakeFlags.Quiet {
		fmt.Printf("%s committed\n", commitID)
	}
	return nil
}
