package inspect

import (
	"context"
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zedzst "github.com/brimdata/zed/cmd/zed/zst"
	zstcmd "github.com/brimdata/zed/cmd/zed/zst"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/zst"
)

var Inspect = &charm.Spec{
	Name:  "inspect",
	Usage: "inspect [flags] file",
	Short: "look at info in a zst file",
	Long: `
The inspect command extracts information from a zst file.
This is mostly useful for test and debug, though there may be interesting
uses as the zst format becomes richer with pruning information and other internal
aggregations about the columns and so forth.

The -R option (on by default) sends the reassembly records to the output while
the -trailer option (off by defaulut) indicates that the trailer should be included.
`,
	New: newCommand,
}

func init() {
	zedzst.Cmd.Add(Inspect)
}

type Command struct {
	*zedzst.Command
	outputFlags outputflags.Flags
	trailer     bool
	reassembly  bool
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zstcmd.Command)}
	f.BoolVar(&c.trailer, "trailer", false, "include the zst trailer in the output")
	f.BoolVar(&c.reassembly, "R", true, "include the zst reassembly section in the output")
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("zst inspect: must be run with a single file argument")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := args[0]
	local := storage.NewLocalEngine()
	reader, err := zst.NewReaderFromPath(ctx, zson.NewContext(), local, path)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	defer func() {
		if writer != nil {
			writer.Close()
		}
	}()
	if c.reassembly {
		r := reader.NewReassemblyReader()
		if err := zio.Copy(writer, r); err != nil {
			return err
		}
	}
	if c.trailer {
		r := reader.NewTrailerReader()
		if err := zio.Copy(writer, r); err != nil {
			return err
		}
	}
	err = writer.Close()
	writer = nil
	return err
}
