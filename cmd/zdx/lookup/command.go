package lookup

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimsec/zq/cmd/zdx/root"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
	"github.com/mccanne/charm"
	"golang.org/x/crypto/ssh/terminal"
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
	key          string
	outputFile   string
	WriterFlags  zio.WriterFlags
	closest      bool
	textShortcut bool
	forceBinary  bool
}

func newLookupCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LookupCommand{Command: parent.(*root.Command)}
	f.StringVar(&c.key, "k", "", "key to search")
	f.BoolVar(&c.closest, "c", false, "find closest insead of exact match")
	f.BoolVar(&c.textShortcut, "t", false, "use format tzng independent of -f option")
	f.BoolVar(&c.forceBinary, "B", false, "allow binary zng be sent to a terminal output")
	c.WriterFlags.SetFlags(f)
	return c, nil
}

func isTerminal(f *os.File) bool {
	return terminal.IsTerminal(int(f.Fd()))
}

func (c *LookupCommand) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zdx lookup: must be run with a single file argument")
	}
	if c.textShortcut {
		c.WriterFlags.Format = "tzng"
	}
	if c.WriterFlags.Format == "zng" && isTerminal(os.Stdout) && !c.forceBinary {
		return errors.New("zq: writing binary zng data to terminal; override with -B or use -t for text.")
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
	defer finder.Close()
	if keyType == nil {
		return fmt.Errorf("%s: index is empty", path)
	}
	// XXX Parse doesn't work yet for record values, but everything else
	// is ready to go to use records and index keys
	key, err := keyType.Parse([]byte(c.key))
	if err != nil {
		return err
	}
	var rec *zng.Record
	if c.closest {
		rec, err = finder.LookupClosest(zng.Value{keyType, key})
	} else {
		rec, err = finder.Lookup(zng.Value{keyType, key})
	}
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
