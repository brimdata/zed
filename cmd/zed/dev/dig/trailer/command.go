package trailer

import (
	"bytes"
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	"github.com/brimdata/zed/cmd/zed/dev/dig"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
)

var Trailer = &charm.Spec{
	Name:  "trailer",
	Usage: "trailer file",
	Short: "read a Zed trailer and output it as Zed",
	Long: `
The trailer command takes a file argument specified by path (which must be a
a sectioned ZNG file with a trailer),
extracts the trailer of the ZNG file, and outputs the section in any Zed format.
`,
	New: New,
}

func init() {
	dig.Cmd.Add(Trailer)
}

type Command struct {
	*dig.Command
	outputFlags outputflags.Flags
}

func MibToBytes(mib float64) int {
	return int(mib * 1024 * 1024)
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*dig.Command)}
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
		return errors.New("zed dev trailer: requires a single file argument")
	}
	uri, err := storage.ParseURI(args[0])
	if err != nil {
		return err
	}
	engine := storage.NewLocalEngine()
	r, err := engine.Get(ctx, uri)
	if err != nil {
		return err
	}
	defer r.Close()
	size, err := storage.Size(r)
	if err != nil {
		return err
	}
	b, err := zngio.ReadTrailerAsBytes(r, size)
	if err != nil {
		return err
	}
	writer, err := c.outputFlags.Open(ctx, engine)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, zngio.NewReader(bytes.NewReader(b), zed.NewContext())); err != nil {
		return err
	}
	return writer.Close()
}
