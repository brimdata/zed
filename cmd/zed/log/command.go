package log

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli"
	"github.com/brimdata/zed/cli/lakeflags"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
)

var Cmd = &charm.Spec{
	Name:  "log",
	Usage: "log [options]",
	Short: "display the commit log history starting at any commit",
	Long: `
The log command outputs a commit history of any branch or unnamed commit object
from a data pool in the format desired.
By default, the output is in the human-readable "lake" format
but ZNG can be used to easily be pipe a log to zq or other tooling for analysis.
`,
	New: New,
}

type Command struct {
	*root.Command
	cli.LakeFlags
	outputFlags outputflags.Flags
	lakeFlags   lakeflags.Flags
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.outputFlags.DefaultFormat = "lake"
	c.outputFlags.SetFlags(f)
	c.LakeFlags.SetFlags(f)
	c.lakeFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 0 {
		return errors.New("no arguments allowed")
	}
	lake, err := c.Open(ctx)
	if err != nil {
		return err
	}
	head, err := c.lakeFlags.HEAD()
	if err != nil {
		return err
	}
	query, err := head.FromSpec("log")
	if err != nil {
		return err
	}
	if c.outputFlags.Format == "lake" {
		c.outputFlags.WriterOpts.Lake.Head = head.Branch
	}
	w, err := c.outputFlags.Open(ctx, storage.NewLocalEngine())
	if err != nil {
		return err
	}
	defer w.Close()
	q, err := lake.Query(ctx, nil, query)
	if err != nil {
		return err
	}
	err = zio.Copy(w, q)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	return err
}
