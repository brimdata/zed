package create

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

// XXX TBD: allow zql expression to combine a same-key value stream
// XXX TBD: right now, this command only takes bzng input but it should
// handle anything zq can take (e.g., json with type configs)... it's easy
// enough to run zq and pipe it to 'zdx create', but it would be nice to
// re-factor zq's input machinery into a separate package so it can be
// re-used here (and by zar).

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-f framesize] [ -o file ] [-value field] -key field bzng-file",
	Short: "generate an zdx file from one or more zng files",
	Long: `
The create command generates an zdx bundle containing of keys and optional values
from the input bzng file.  The required option f -k specifies the zng record
field name that comprises the set of keys added to the zdx.  If a value field is
specified with -value, then that field specifies the values to include with each key.
If a key appears more than once, the last value in the input takes precendence.
It is an error for a value field is specified but not present in any record.
It is also an error if the key or value fields are not of uniform type.`,
	New: newCreateCommand,
}

func init() {
	root.Zdx.Add(Create)
}

type CreateCommand struct {
	*root.Command
	framesize  int
	outputFile string
	keyField   string
	valField   string
	skip       bool
}

func newCreateCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &CreateCommand{
		Command: parent.(*root.Command),
	}
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in zdx file")
	f.StringVar(&c.outputFile, "o", "zdx", "output zdx bundle name")
	f.StringVar(&c.keyField, "k", "", "field name of keys ")
	f.StringVar(&c.valField, "v", "", "field name of values ")
	f.BoolVar(&c.skip, "S", false, "skip all records except for the first of each stream")
	return c, nil
}

func (c *CreateCommand) Run(args []string) error {
	if c.keyField == "" {
		return errors.New("must specify a key field with -key")
	}
	if len(args) != 1 {
		return errors.New("must specify a single bzng input file containing keys and optional values")
	}
	readKey := expr.CompileFieldAccess(c.keyField)
	var readVal expr.FieldExprResolver
	if c.valField != "" {
		readVal = expr.CompileFieldAccess(c.valField)
	}
	path := args[0]
	var in io.Reader
	if path == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}
	zctx := resolver.NewContext()
	reader := bzngio.NewReader(in, zctx)
	writer, err := zdx.NewWriter(c.outputFile, c.framesize)
	if err != nil {
		return err
	}
	defer writer.Close()
	table := zdx.NewMemTable(zctx)
	read := reader.Read
	if c.skip {
		// to skip, return the first record of each stream,
		// meaning read the first one, then skip to the next
		// sos for each subsequent read
		read = func() (*zng.Record, error) {
			if reader.Position() == 0 {
				return reader.Read()
			}
			rec, _, err := reader.SkipStream()
			return rec, err
		}
	}
	for {
		rec, err := read()
		if err != nil {
			return err
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
			// the right thing to odo.
			continue
		}
		if readVal == nil {
			if err := table.EnterKey(k); err != nil {
				return err
			}
		} else {
			v := readVal(rec)
			if v.Type == nil {
				// the key field exists but the value field
				// doesn't, bail with an error
				return fmt.Errorf("couldn't read value '%s' (%s)", c.valField, rec)
			}
			// XXX here is where the table could be configured
			// with a reducer to coalesce values that land
			// on the same key.  right now, a new value
			// will clobber the old one.
			if err := table.EnterKeyVal(k, v); err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}
	return zbuf.Copy(writer, table)
}
