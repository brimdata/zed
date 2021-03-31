package convert

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/cli/inputflags"
	zedindex "github.com/brimdata/zed/cmd/zed/index"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/detector"
	"github.com/brimdata/zed/zng/resolver"
)

var Convert = &charm.Spec{
	Name:  "convert",
	Usage: "convert [-f frametresh] [ -o file ] -k field[,field,...] file",
	Short: "generate a zed index file from one or more zng files",
	Long: `
The convert command generates a zed index containing keys and optional values
from the input file.  The required flag -k specifies one or more zng record
field names that comprise the index search keys, in precedence order.
The keys must be pre-sorted in ascending order with
respect to the stream of zng records; otherwise the index will not work correctly.
The input records are all copied to the base layer of the output index, as is,
so any information stored alongside the keys (e.g., pre-computed aggregations).
It is an error if the key or value fields are not of uniform type.`,
	New: newCommand,
}

func init() {
	zedindex.Cmd.Add(Convert)
}

type Command struct {
	*zedindex.Command
	frameThresh int
	desc        bool
	outputFile  string
	keys        string
	inputFlags  inputflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*zedindex.Command),
	}
	f.IntVar(&c.frameThresh, "f", 32*1024, "minimum frame size used in zed index file")
	f.BoolVar(&c.desc, "desc", false, "specify data is in descending order")
	f.StringVar(&c.outputFile, "o", "index.zng", "name of index output file")
	f.StringVar(&c.keys, "k", "", "comma-separated list of field names for keys")
	c.inputFlags.SetFlags(f)

	return c, nil
}

func (c *Command) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(&c.inputFlags); err != nil {
		return err
	}
	if c.keys == "" {
		return errors.New("must specify at least one key field with -k")
	}
	//XXX no reason to limit this
	if len(args) != 1 {
		return errors.New("must specify a single zng input file containing keys and optional values")
	}
	path := args[0]
	if path == "-" {
		path = iosrc.Stdin
	}
	zctx := resolver.NewContext()
	file, err := detector.OpenFile(zctx, path, c.inputFlags.Options())
	if err != nil {
		return err
	}
	defer file.Close()
	writer, err := index.NewWriter(zctx, c.outputFile,
		index.KeyFields(field.DottedList(c.keys)...),
		index.FrameThresh(c.frameThresh),
		index.Order(zbuf.Order(c.desc)),
	)
	if err != nil {
		return err
	}
	if err := zbuf.Copy(writer, zbuf.Reader(file)); err != nil {
		writer.Close()
		return err
	}
	return writer.Close()
}
