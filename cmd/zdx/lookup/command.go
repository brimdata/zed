package lookup

import (
	"errors"
	"flag"
	"fmt"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio"
	"github.com/mccanne/charm"
)

var Lookup = &charm.Spec{
	Name:  "lookup",
	Usage: "lookup -k <key> <bundle>",
	Short: "lookup a key in an zdx file and print value as zng record",
	Long: `
The lookup command locates the specified <key> in the base file of the
zdx <bundle> and displays the result as a zng record.
The key argument specifies a value to look up in the table and must be parseable
as a zng type of the key that was originally indexed.`,
	New: newLookupCommand,
}

func init() {
	root.Zdx.Add(Lookup)
}

type LookupCommand struct {
	*root.Command
	key         string
	outputFile  string
	WriterFlags zio.WriterFlags
}

func newLookupCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LookupCommand{Command: parent.(*root.Command)}
	f.StringVar(&c.key, "k", "", "key to search")
	c.WriterFlags.SetFlags(f)
	return c, nil
}

func (c *LookupCommand) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zdx lookup: must be run with a single file argument")
	}
	path := args[0]
	if c.key == "" {
		return errors.New("must specify a key")
	}
	finder := zdx.NewFinder(path)
	keyType, err := finder.Open()
	if err != nil {
		return err
	}
	if keyType == nil {
		return fmt.Errorf("%s: index is empty", path)
	}
	defer finder.Close()
	// XXX Parse doesn't work yet for record values, but everything else
	// is ready to go to use records and index keys
	key, err := keyType.Parse([]byte(c.key))
	if err != nil {
		return err
	}
	rec, err := finder.Lookup(key)
	if err != nil {
		return err
	}
	if rec == nil {
		return nil
	}
	writer, err := emitter.NewFile(c.outputFile, &c.WriterFlags)
	if err != nil {
		return err
	}
	if err := writer.Write(rec); err != nil {
		return err
	}
	return writer.Close()
}
