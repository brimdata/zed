package read

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	devvcache "github.com/brimdata/zed/cmd/zed/dev/vcache"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

var Project = &charm.Spec{
	Name:  "project",
	Usage: "project [flags] field[,field...] path",
	Short: "read a ZST file and run a projection as a test",
	Long: `
The project command reads ZST vectors from
a ZST storage objects (local files or s3 objects) and outputs
the reconstructed ZNG row data as a projection of one or more fields.

This command is most useful for testing the ZST vector cache.
`,
	New: newCommand,
}

func init() {
	devvcache.Cmd.Add(Project)
}

type Command struct {
	*root.Command
	outputFlags outputflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) < 2 {
		return errors.New("zst read: must be run with a single path argument followed by one or more fields")
	}
	uri, err := storage.ParseURI(args[0])
	if err != nil {
		return err
	}
	fields := args[1:]
	local := storage.NewLocalEngine()
	cache := vcache.NewCache(local)
	object, err := cache.Fetch(ctx, uri, ksuid.Nil)
	if err != nil {
		return err
	}
	defer object.Close()
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	projection, err := object.NewProjection(fields)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, projection); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
