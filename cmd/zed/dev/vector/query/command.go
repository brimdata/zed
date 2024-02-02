package query

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/dev/vector"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/vcache"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

var query = &charm.Spec{
	Name:  "query",
	Usage: "query [flags] query path",
	Short: "run a Zed query on a VNG file",
	Long: `
The query command runs a query on a VNG file presuming the 
query is entirely vectorizable.  The VNG object is read through 
the vcache and projected as needed into the runtime.

This command is most useful for testing the vector runtime
in isolation from a Zed lake.
`,
	New: newCommand,
}

func init() {
	vector.Cmd.Add(query)
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
		return errors.New("usage: query followed by a single path argument of VNG data")
	}
	text := args[0]
	uri, err := storage.ParseURI(args[1])
	if err != nil {
		return err
	}
	local := storage.NewLocalEngine()
	cache := vcache.NewCache(local)
	object, err := cache.Fetch(ctx, uri, ksuid.Nil)
	if err != nil {
		return err
	}
	defer object.Close()
	rctx := runtime.NewContext(ctx, zed.NewContext())
	puller, err := compiler.VectorCompile(rctx, text, object)
	if err != nil {
		return err
	}
	writer, err := c.outputFlags.Open(ctx, local)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, zbuf.PullerReader(puller)); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
