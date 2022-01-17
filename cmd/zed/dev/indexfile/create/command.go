package create

import (
	"errors"
	"flag"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/inputflags"
	"github.com/brimdata/zed/cmd/zed/dev/indexfile"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-f frameThresh] [ -o file ] -k field file",
	Short: "create a key-only Zed index from a zng file",
	Long: `
The create command generates a key-only Zed index file comprising the values from the
input taken from the field specified by -k.  The output index will have a base layer
with search key called "key".
If a key appears more than once, the last value in the input takes precedence.
It is an error if the key fields are not of uniform type.`,
	New: newCommand,
}

func init() {
	indexfile.Cmd.Add(Create)
}

type Command struct {
	*indexfile.Command
	frameThresh int
	outputFile  string
	keys        field.List
	skip        bool
	inputReady  bool
	inputFlags  inputflags.Flags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*indexfile.Command)}
	f.IntVar(&c.frameThresh, "f", 32*1024, "minimum frame size used in index file")
	f.StringVar(&c.outputFile, "o", "index.zng", "name of index output file")
	f.Func("k", "field name of search keys", c.parseKey)
	f.BoolVar(&c.inputReady, "x", false, "input file is already sorted keys (and optional values)")
	f.BoolVar(&c.skip, "S", false, "skip all records except for the first of each stream")
	c.inputFlags.SetFlags(f, true)
	return c, nil
}

func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.inputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(c.keys) == 0 {
		return errors.New("must specify at least one key field with -k")
	}
	//XXX no reason to limit this... we will fix this when we refactor
	// the code here to use Zed/proc instead for the hash table (after we
	// have spillable group-bys)
	if len(args) != 1 {
		return errors.New("must specify a single zng input file containing the indicated keys")
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
	writer, err := index.NewWriter(zctx, local, c.outputFile, c.keys, index.FrameThresh(c.frameThresh))
	if err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			writer.Close()
		}
	}()
	reader, err := c.buildTable(zctx, file)
	if err != nil {
		return err
	}
	if err := zio.Copy(writer, reader); err != nil {
		return err
	}
	close = false
	return writer.Close()
}

func (c *Command) parseKey(s string) error {
	c.keys = append(c.keys, field.Dotted(s))
	return nil
}

func (c *Command) buildTable(zctx *zed.Context, reader zio.Reader) (*index.MemTable, error) {
	fields, resolvers := compiler.CompileAssignments(zctx, c.keys, c.keys)
	cutter, err := expr.NewCutter(zctx, fields, resolvers)
	if err != nil {
		return nil, err
	}
	table := index.NewMemTable(zctx, c.keys)
	ectx := expr.NewContext()
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		k := cutter.Eval(ectx, rec)
		if k.IsError() {
			// if the key doesn't exist, just skip it
			continue
		}
		if k.IsNull() {
			// The key field is null.  Skip it.  Unless we want to
			// index nulls, this is the right thing to do.
			continue
		}
		if err := table.Enter(k); err != nil {
			return nil, err
		}
	}
	return table, nil
}
