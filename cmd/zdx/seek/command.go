package seek

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
)

var Seek = &charm.Spec{
	Name:  "seek",
	Usage: "seek [-f framesize] [ -o file ] [-v field] -k field file",
	Short: "generate a seek-style index file for a zng file",
	Long: `
The seek command creates an index for the seek offsets of each
start-of-stream (sos) in a zng file.  The key field is specified by -k and all
values in this field must be in ascending order.  The seek offset of each sos
is stored as the field "offset" in the base layer of the output search index
unless overridden by -v.
It is an error if the values of the key field are not of uniform type.

This is command is useful for creating time indexes for large zng logs where the
zng records are sorted by time.`,
	New: newCommand,
}

func init() {
	root.Zdx.Add(Seek)
}

type Command struct {
	*root.Command
	framesize   int
	outputFile  string
	keyField    string
	offsetField string
	ReaderFlags zio.ReaderFlags
}

func newCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{
		Command: parent.(*root.Command),
	}
	f.IntVar(&c.framesize, "f", 32*1024, "minimum frame size used in zdx file")
	f.StringVar(&c.outputFile, "o", "zdx", "output index name")
	f.StringVar(&c.keyField, "k", "", "name of search key field")
	f.StringVar(&c.offsetField, "v", "offset", "field name for seek offset in output index")
	c.ReaderFlags.SetFlags(f)

	return c, nil
}

func (c *Command) Run(args []string) error {
	//XXX no reason to limit this... fix later
	if len(args) != 1 {
		return errors.New("must specify a single zng input file containing keys and optional values")
	}
	file := os.Stdin
	path := args[0]
	if path != "-" {
		var err error
		file, err = os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
	}
	zctx := resolver.NewContext()
	reader := zngio.NewReader(file, zctx)
	writer, err := zdx.NewWriter(zctx, c.outputFile, []string{c.keyField}, c.framesize)
	if err != nil {
		return err
	}
	close := true
	defer func() {
		if close {
			writer.Close()
		}
	}()
	readKey := expr.CompileFieldAccess(c.keyField)
	var builder *zng.Builder
	var keyType zng.Type
	var offset int64
	// to skip to each sos, we read the first rec normally
	// then call SkipStream and the bottmo of the for-loop.
	rec, err := reader.Read()
	for err == nil && rec != nil {
		k := readKey(rec)
		if k.Type == nil || k.Bytes == nil {
			// if the key doesn't exist or is unset, fail here
			// XXX we should check that key order is ascending
			return fmt.Errorf("key field is missing: %s", rec)
		}
		if builder == nil {
			keyType = k.Type
			cols := []zng.Column{
				{c.keyField, k.Type},
				{c.offsetField, zng.TypeInt64},
			}
			typ, err := zctx.LookupTypeRecord(cols)
			if err != nil {
				return err
			}
			builder = zng.NewBuilder(typ)
		} else if keyType != k.Type {
			return fmt.Errorf("key type changed from %s to %s", keyType, k.Type)
		}
		offBytes := zng.EncodeInt(offset)
		out := builder.Build(k.Bytes, offBytes)
		if err := writer.Write(out); err != nil {
			return err
		}
		rec, offset, err = reader.SkipStream()
	}
	if err != nil {
		return err
	}
	// We do this little song and dance so we can return error on close
	// but don't call close twice if we make it here.
	close = false
	return writer.Close()
}
