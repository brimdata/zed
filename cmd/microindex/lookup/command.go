package lookup

import (
	"context"
	"errors"
	"flag"
	"os"
	"strings"

	"github.com/brimsec/zq/cmd/microindex/root"
	"github.com/brimsec/zq/emitter"
	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/mccanne/charm"
	"golang.org/x/crypto/ssh/terminal"
)

var Lookup = &charm.Spec{
	Name:  "lookup",
	Usage: "lookup -k key[,key...] index",
	Short: "lookup a key in a microindex file and print value as zng record",
	Long: `
The lookup command locates the specified key(s) in the base layer of a
microindex and displays the result as a zng record.
If the index has multiple keys, then multiple records may be returned for
all the records that match the supplied keys.
Each key argument specifies a value to look up in the table and must be parseable
as the zng type of the key that was originally indexed where the keys refer to the leaf
values in left-to-right order of the keys represented as a record, inclusive
of any nested records.`,
	New: newLookupCommand,
}

func init() {
	root.MicroIndex.Add(Lookup)
}

type LookupCommand struct {
	*root.Command
	keys         string
	outputFile   string
	WriterFlags  zio.WriterFlags
	closest      bool
	textShortcut bool
	forceBinary  bool
}

func newLookupCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LookupCommand{Command: parent.(*root.Command)}
	f.StringVar(&c.keys, "k", "", "key(s) to search")
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
		return errors.New("microindex lookup: must be run with a single file argument")
	}
	if c.textShortcut {
		c.WriterFlags.Format = "tzng"
	}
	if c.WriterFlags.Format == "zng" && isTerminal(os.Stdout) && !c.forceBinary {
		return errors.New("writing binary zng data to terminal; override with -B or use -t for text.")
	}
	path := args[0]
	if c.keys == "" {
		return errors.New("must specify one or more comma-separated keys")
	}
	uri, err := iosrc.ParseURI(path)
	if err != nil {
		return err
	}
	finder := microindex.NewFinder(resolver.NewContext(), uri)
	if err := finder.Open(); err != nil {
		return err
	}
	defer finder.Close()
	keys, err := finder.ParseKeys(strings.Split(c.keys, ","))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hits := make(chan *zng.Record)
	var searchErr error
	go func() {
		if c.closest {
			var rec *zng.Record
			rec, searchErr = finder.LookupClosest(keys)
			if rec != nil {
				hits <- rec
			}
		} else {
			searchErr = finder.LookupAll(ctx, hits, keys)
		}
		close(hits)
	}()
	writer, err := emitter.NewFile(c.outputFile, &c.WriterFlags)
	if err != nil {
		return err
	}
	for hit := range hits {
		if err := writer.Write(hit); err != nil {
			writer.Close()
			return err
		}
	}
	err = writer.Close()
	if searchErr != nil {
		err = searchErr
	}
	return err
}
