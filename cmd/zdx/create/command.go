package create

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

// XXX TBD: allow zql expression to combine a same-key value stream

var Create = &charm.Spec{
	Name:  "create",
	Usage: "create [-f framesize] [ -o file ] [-value field] -key field file",
	Short: "generate an zdx file from one or more zng files",
	Long: `
The create command generates a zdx bundle containing keys and optional values
from the input file.  The required flag -k specifies the zng record
field name that comprises the set of keys added to the zdx.  The optional
flag -v specifies a field name whose value will be added alongside its key.
If a key appears more than once, the last value in the input takes precedence.
It is an error if a value field is specified but not present in any record.
It is also an error if the key or value fields are not of uniform type.`,
	New: newCreateCommand,
}

func init() {
	root.Zdx.Add(Create)
}

type CreateCommand struct {
	*root.Command
	framesize   int
	outputFile  string
	keyField    string
	valField    string
	skip        bool
	ReaderFlags zio.ReaderFlags
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
	c.ReaderFlags.SetFlags(f)

	return c, nil
}

func (c *CreateCommand) Run(args []string) error {
	if c.keyField == "" {
		return errors.New("must specify a key field with -k")
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
	if path == "-" {
		path = "" // stdin... this will change
	}
	zctx := resolver.NewContext()
	reader, err := detector.OpenFile(zctx, path, &c.ReaderFlags)
	if err != nil {
		return err
	}
	writer, err := zdx.NewWriter(c.outputFile, c.framesize)
	if err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			writer.Close()
		}
	}()
	table := zdx.NewMemTable(zctx)
	read := reader.Read
	if c.skip {
		reader, ok := reader.Reader.(*bzngio.Reader)
		if !ok {
			return errors.New("cannot use -S flag with non-bzng input")
		}
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
			// the right thing to do.
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
	close = false
	if err := zbuf.Copy(writer, table); err != nil {
		return err
	}
	return writer.Close()
}
