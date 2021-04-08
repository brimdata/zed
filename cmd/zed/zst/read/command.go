package inspect

import (
	"context"
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	zstcmd "github.com/brimdata/zed/cmd/zed/zst"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/zst"
)

var Read = &charm.Spec{
	Name:  "read",
	Usage: "read [flags] path",
	Short: "read a zst file and output as zng",
	Long: `
The read command reads columnar zst from
a zst storage objects (local files or s3 objects) and outputs
the reconstructed zng row data in the format of choice.

This command is most useful for test, debug, and demo as you can also
read zst objects with zq.
`,
	New: newCommand,
}

func init() {
	zstcmd.Cmd.Add(Read)
}

type Command struct {
	*zstcmd.Command
	outputFlags outputflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*zstcmd.Command)}
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.outputFlags); err != nil {
		return err
	}
	if len(args) != 1 {
		return errors.New("zst read: must be run with a single path argument")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := args[0]
	reader, err := zst.NewReaderFromPath(ctx, zson.NewContext(), path)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := c.outputFlags.Open(ctx)
	if err != nil {
		return err
	}
	if err := zbuf.Copy(writer, reader); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
