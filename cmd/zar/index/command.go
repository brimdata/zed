package index

import (
	"errors"
	"flag"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/cmd/zar/root"
	"github.com/mccanne/charm"
)

var Index = &charm.Spec{
	Name:  "index",
	Usage: "index dir",
	Short: "creates index files for bzng files",
	Long: `
zar find descends the directory argument looking for bzng files and creates an index
file for IP addresses for each bzng file encountered.  An index is written to
a sub-directory of the directory containing each encountered bzng file, where the
name of the sub-directory is a concatenation of the bzng file name and the suffix
".zar".
The current version supports only IP address, but this will soon change.
`,
	New: New,
}

func init() {
	root.Zar.Add(Index)
}

type Command struct {
	*root.Command
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) != 1 {
		return errors.New("zar index: exactly one directory must be specified")
	}
	dir := args[0]
	return archive.CreateIndexes(dir)
}
