package run

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/dev/vector"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/vam"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

var Agg = &charm.Spec{
	Name:  "run",
	Usage: "run [flags] query path",
	Short: "run a Zed query on a VNG file",
	Long: `
The run command runs a query on a VNG file presuming the 
query is entirely vectorizable.  The VNG object is read through 
the vcache and projected as needed into the runtime.

This command is most useful for testing the vector runtime
in isolation from a Zed lake.
`,
	New: newCommand,
}

func init() {
	vector.Cmd.Add(Agg)
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
	if len(args) != 2 {
		return errors.New("requires a query followed by a single path argument of the VNG data")
	}
	uri, err := storage.ParseURI(args[0])
	if err != nil {
		return err
	}
	text := args[0]
	uri := args[1]
	local := storage.NewLocalEngine()
	cache := vcache.NewCache(local)
	object, err := cache.Fetch(ctx, uri, ksuid.Nil)
	if err != nil {
		return err
	}
	defer object.Close()
	// Make a projection to act as the source of query using the
	// query's demand to narrow the vectors read.
	projection := vam.NewProjection(zed.NewContext(), object, paths)
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}

	//XXX
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, zbuf.PullerReader(vam.NewMaterializer(agg))); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
