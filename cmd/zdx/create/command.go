package create

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-f framesize] [ -o file ] -k field file",
	Short: "create a key-only zdx index from a zng file",
	Long: `
The create command generates a key-only zdx index comprising the values from the
input taken from the field specified by -k.  The output index will have a base layer
with search key called "key".
If a key appears more than once, the last value in the input takes precedence.
It is an error if the key fields are not of uniform type.`,
	New: newCommand,
}

func init() {
	root.Zdx.Add(Create)
}

type Command struct {
	*root.Command
	framesize   int
	outputFile  string
	keyField    string
	skip        bool
	inputReady  bool
	ReaderFlags zio.ReaderFlags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*root.Command),
	}
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in zdx file")
	f.StringVar(&c.outputFile, "o", "zdx", "output zdx bundle name")
	f.StringVar(&c.keyField, "k", "", "field name of search keys")
	f.BoolVar(&c.inputReady, "x", false, "input file is already sorted keys (and optional values)")
	f.BoolVar(&c.skip, "S", false, "skip all records except for the first of each stream")
	c.ReaderFlags.SetFlags(f)

	return c, nil
}

func (c *Command) Run(args []string) error {
	if c.keyField == "" {
		return errors.New("must specify at least one key field with -k")
	}
	//XXX no reason to limit this... we will fix this when we refactor
	// the code here to use zql/proc instead fo the hash table (after we
	// have spillable group-bys)
	if len(args) != 1 {
		return errors.New("must specify a single zng input file containing the indicated keys")
	}
	path := args[0]
	zctx := resolver.NewContext()
	cfg := detector.OpenConfig{
		Format:    c.ReaderFlags.Format,
		DashStdin: true,
		//JSONTypeConfig: c.jsonTypeConfig,
		//JSONPathRegex:  c.jsonPathRegexp,
	}
	file, err := detector.OpenFile(zctx, path, cfg)
	if err != nil {
		return err
	}
	writer, err := zdx.NewWriter(zctx, c.outputFile, nil, c.framesize)
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
	if err := zbuf.Copy(writer, reader); err != nil {
		return err
	}
	close = false
	return writer.Close()
}

func (c *Command) buildTable(zctx *resolver.Context, reader zbuf.Reader) (*zdx.MemTable, error) {
	readKey := expr.CompileFieldAccess(c.keyField)
	table := zdx.NewMemTable(zctx)
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		k := readKey(rec)
		if k.Type == nil {
			// if the key doesn't exist, just skip it
			continue
		}
		if k.Bytes == nil {
			// The key field is unset.  Skip it.  Unless we want to
			// index the notion of something that is unset, this is
			// the right thing to do.
			continue
		}
		if err := table.EnterKey(k); err != nil {
			return nil, err
		}
	}
	return table, nil
}
