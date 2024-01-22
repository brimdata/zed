package read

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	devvcache "github.com/brimdata/zed/cmd/zed/dev/vcache"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/vam"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

var Project = &charm.Spec{
	Name:  "project",
	Usage: "project [flags] path [field ...]",
	Short: "read a VNG file and run a projection as a test",
	Long: `
The project command reads VNG vectors from
VNG storage objects (local files or s3 objects) and outputs
the reconstructed ZNG row data as a projection of zero or more fields.
If no fields are specified, all the data is projected.

This command is most useful for testing the VNG vector cache.
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
		return errors.New("VNG read: must be run with a single path argument followed by one or more fields")
	}
	uri, err := storage.ParseURI(args[0])
	if err != nil {
		return err
	}
	var paths []field.Path
	for _, dotted := range args[1:] {
		paths = append(paths, field.Dotted(dotted))
	}
	local := storage.NewLocalEngine()
	cache := vcache.NewCache(local)
	object, err := cache.Fetch(ctx, uri, ksuid.Nil)
	if err != nil {
		return err
	}
	defer object.Close()
	projection := vam.NewProjection(zed.NewContext(), object, paths)
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	if err := zbuf.CopyPuller(writer, projection); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
