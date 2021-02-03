package convert

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/cli/inputflags"
	"github.com/brimsec/zq/cmd/microindex/root"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Convert = &charm.Spec{
	Name:  "convert",
	Usage: "convert [-f frametresh] [ -o file ] -k field[,field,...] file",
	Short: "generate a microindex file from one or more zng files",
	Long: `
The convert command generates a microindex containing keys and optional values
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
	root.MicroIndex.Add(Convert)
}

type Command struct {
	*root.Command
	frameThresh int
	desc        bool
	outputFile  string
	keys        string
	inputFlags  inputflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*root.Command),
	}
	f.IntVar(&c.frameThresh, "f", 32*1024, "minimum frame size used in microindex file")
	f.BoolVar(&c.desc, "desc", false, "specify data is in descending order")
	f.StringVar(&c.outputFile, "o", "index.zng", "name of microindex output file")
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
	writer, err := microindex.NewWriter(zctx, c.outputFile,
		microindex.KeyFields(field.DottedList(c.keys)...),
		microindex.FrameThresh(c.frameThresh),
		microindex.Order(zbuf.Order(c.desc)),
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
