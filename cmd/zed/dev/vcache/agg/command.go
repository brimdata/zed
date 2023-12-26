package agg

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/outputflags"
	devvcache "github.com/brimdata/zed/cmd/zed/dev/vcache"
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
	Name:  "agg",
	Usage: "agg [flags] field[,field...] path",
	Short: "read a VNG file and run an aggregate as a test",
	Long: `
The project command reads VNG vectors from
a VNG storage objects (local files or s3 objects) and outputs
the reconstructed ZNG row data as an aggregate function.

This command is most useful for testing the VNG vector cache.
`,
	New: newCommand,
}

func init() {
	devvcache.Cmd.Add(Agg)
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
		//XXX
		return errors.New("VNG read: must be run with a single path argument followed by one or more fields")
	}
	uri, err := storage.ParseURI(args[0])
	if err != nil {
		return err
	}
	field := args[1]
	local := storage.NewLocalEngine()
	cache := vcache.NewCache(local)
	object, err := cache.Fetch(ctx, uri, ksuid.Nil)
	if err != nil {
		return err
	}
	defer object.Close()
	//XXX nil puller
	agg := vam.NewCountByString(object.LocalContext(), nil, field)
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
