package read

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	devzst "github.com/brimdata/zed/cmd/zed/dev/zst"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
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
	devzst.Cmd.Add(Read)
}

type Command struct {
	*devzst.Command
	outputFlags outputflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*devzst.Command)}
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("zst read: must be run with a single path argument")
	}
	path := args[0]
	local := storage.NewLocalEngine()
	reader, err := zst.NewReaderFromPath(ctx, zed.NewContext(), local, path)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, reader); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
