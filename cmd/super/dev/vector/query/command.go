package query

import (
	"errors"
	"flag"

	zed "github.com/brimdata/super"
	"github.com/brimdata/super/cli/outputflags"
	"github.com/brimdata/super/cli/queryflags"
	"github.com/brimdata/super/cmd/super/dev/vector"
	"github.com/brimdata/super/cmd/super/root"
	"github.com/brimdata/super/compiler"
	"github.com/brimdata/super/pkg/charm"
	"github.com/brimdata/super/pkg/storage"
	"github.com/brimdata/super/runtime"
	"github.com/brimdata/super/runtime/vcache"
	"github.com/brimdata/super/zbuf"
	"github.com/brimdata/super/zio"
	"github.com/segmentio/ksuid"
)

var spec = &charm.Spec{
	Name:  "query",
	Usage: "query [flags] query path",
	Short: "run a query on a VNG file",
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
	vector.Spec.Add(spec)
}

type Command struct {
	*root.Command
	outputFlags outputflags.Flags
	queryFlags  queryflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	c.outputFlags.SetFlags(f)
	c.queryFlags.SetFlags(f)
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
	puller, err := compiler.VectorCompile(rctx, c.queryFlags.SQL, text, object)
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
