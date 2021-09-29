package lookup

import (
	"errors"
	"flag"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/cli/outputflags"
	zedindex "github.com/brimdata/zed/cmd/zed/index"
	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zson"
)

var Lookup = &charm.Spec{
	Name:  "lookup",
	Usage: "lookup -k key[,key...] index",
	Short: "lookup a key in a zed index file and print value as zng record",
	Long: `
The lookup command locates the specified key(s) in the base layer of a
zed index file and displays the result as a zng record.
If the index has multiple keys, then multiple records may be returned for
all the records that match the supplied keys.
Each key argument specifies a value to look up in the table and must be parseable
as the zng type of the key that was originally indexed where the keys refer to the leaf
values in left-to-right order of the keys represented as a record, inclusive
of any nested records.`,
	New: newLookupCommand,
}

func init() {
	zedindex.Cmd.Add(Lookup)
}

type LookupCommand struct {
	*zedindex.Command
	keys        string
	outputFlags outputflags.Flags
	closest     bool
}

func newLookupCommand(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &LookupCommand{Command: parent.(*zedindex.Command)}
	f.StringVar(&c.keys, "k", "", "key(s) to search")
	f.BoolVar(&c.closest, "c", false, "find closest insead of exact match")
	c.outputFlags.SetFlags(f)
	return c, nil
}

func (c *LookupCommand) Run(args []string) error {
	ctx, cleanup, err := c.Init(&c.outputFlags)
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) != 1 {
		return errors.New("zed index lookup: must be run with a single file argument")
	}
	path := args[0]
	if c.keys == "" {
		return errors.New("must specify one or more comma-separated keys")
	}
	uri, err := storage.ParseURI(path)
	if err != nil {
		return err
	}
	local := storage.NewLocalEngine()
	finder, err := index.NewFinder(ctx, zson.NewContext(), local, uri)
	if err != nil {
		return err
	}
	defer finder.Close()
	keys, err := finder.ParseKeys(strings.Split(c.keys, ",")...)
	if err != nil {
		return err
	}
	hits := make(chan *zed.Record)
	var searchErr error
	go func() {
		if c.closest {
			var rec *zed.Record
			rec, searchErr = finder.ClosestLTE(keys)
			if rec != nil {
				hits <- rec
			}
		} else {
			searchErr = finder.LookupAll(ctx, hits, keys)
		}
		close(hits)
	}()
	writer, err := c.outputFlags.Open(ctx, local)
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
