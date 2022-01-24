package indexfile

import (
	"flag"

	"github.com/brimdata/zed/cmd/zed/dev"
	"github.com/brimdata/zed/cmd/zed/root"
	"github.com/brimdata/zed/pkg/charm"
)

var Cmd = &charm.Spec{
	Name:  "indexfile",
	Usage: "indexfile <command> [options] [arguments...]",
	Short: "create and search Zed indexes",
	Long: `
"zed dev indexfile" is command-line utility for creating and manipulating Zed indexes.

A Zed index is a sectioned ZNG file organized around one or more search keys.
The values in the first section are Zed records that are searchable according
to their key field(s) and are are sorted in either ascending
or descending order as determined when the file is created.
The records are organized into a sequence of indexable chunks,
as support by ZNG end-of-streams.

The first section is followed by one or more additional sections comprising
the hierarchy of a constant B-Tree whose order is derived from the sort key(s)
specified.

A Zed trailer provides the sizes of all the section, the keys, the sort order of the keys.

Since a Zed index is just a ZNG file, its contents can be easily examined
with "zq" or with the "section" and "trailer" sub-commands of "zed dev dig".

The "zed indexfile create" command generates Zed indexes from input data.

The "zed indexfile lookup" command uses a Zed index to location a given value
of the keyed field using the B-Tree.  This command is useful for test and debug
and not typically expected to be used from the command line.  Instead, Zed indexes
are more typically used programatically, e.g., by a Zed lake, to index the
data objects of a large-scale data store.

In this design, each index lookup is not itself particular performant
due to the round-trips required to traverse the B-Tree,
but a large number of parallel index lookups hitting cached portions of
a large index (comprised of many individual Zed indexes) performs quite well in practice.

Note that any type of Zed data can comprise the input: e.g., the target
data to search, the values of an indexed field where the data is stored
elsewhere, pre-computed partials aggregations indexed by group-by key(s),
and so forth.

Note also that the values comprising the keyed fields may be any Zed type and
need not be specified ahead of time.  Since the space of all Zed values has
a total order, an indexed of mixed types has a well-defined order and searches
work without issue across fields of heterogeneous types.

Finally, the input values are assumed to be records which are accesssed to
determine their key value.  If any inputs values are no records, an error
is returned.  (This does not mean that non-record values cannot be indexed;
rather, the use of indexes much use records to represent the searchable
entities.)
`,
	New: New,
}

func init() {
	dev.Cmd.Add(Cmd)
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*root.Command)}, nil
}

func (c *Command) Run(args []string) error {
	_, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	if len(args) == 0 {
		return charm.NeedHelp
	}
	return charm.ErrNoRun
}
