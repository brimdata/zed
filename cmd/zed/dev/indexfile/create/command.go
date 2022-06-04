package create

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cmd/zed/dev/indexfile"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-f frametresh] [ -o file ] -k key[,key,...] file",
	Short: "generate a Zed index file from one or more key-sorted Zed inputs",
	Long: `
The "zed indexfile create" command generates a Zed index from one or more Zed input files
as a sectioned ZNG file with a layout desribed in the "zed indexfile" command.
(Run "zed indexfile -h" for the description.)

The bushiness of the B-tree created is controlled by the -f flag,
which specifies a target size in bytes for each node in the B-tree.

The inputs values are presumed to be presorted by the specified keys
in the order indicated by -order.  If a key is not present in a value,
that value is treated as the null value, and a lookup for null will return
all such values.
`,
	New: newCommand,
}

func init() {
	indexfile.Cmd.Add(Create)
}

type Command struct {
	*indexfile.Command
	opts       index.WriterOpts
	order      string
	outputFile string
	keys       string
	inputFlags inputflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*indexfile.Command)}
	f.IntVar(&c.opts.FrameThresh, "f", 32*1024, "minimum frame size used in Zed index file")
	f.Func("order", "order of index (asc or desc) (default asc)", func(s string) (err error) {
		if s != "" {
			c.opts.Order, err = order.Parse(s)
		}
		return err
	})
	f.StringVar(&c.outputFile, "o", "index.zng", "name of index output file")
	f.StringVar(&c.keys, "k", "", "comma-separated list of field names for keys")
	c.inputFlags.SetFlags(f, true)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.inputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if c.keys == "" {
		return errors.New("must specify at least one key field with -k")
	}
	//XXX no reason to limit this
	if len(args) != 1 {
		return errors.New("must specify a single ZNG input file containing keys and optional values")
	}
	path := args[0]
	if path == "-" {
		path = "stdio:stdin"
	}
	zctx := zed.NewContext()
	local := storage.NewLocalEngine()
	file, err := anyio.Open(ctx, zctx, local, path, c.inputFlags.Options())
	if err != nil {
		return err
	}
	defer file.Close()
	writer, err := index.NewWriter(zctx, local, c.outputFile, field.DottedList(c.keys), c.opts)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, zio.Reader(file)); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
