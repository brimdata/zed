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
	Usage: "index -d dir pattern [ pattern ...]",
	Short: "create zdx index bundles for bzng files",
	Long: `
zar index descends the directory argument starting at dir and looks
for bzng files.  Each bzng file fund is indexed according to the one or
more indexing rules provided.

A pattern is either a field name or a ":" followed by a zng type name.
For example, to index the all fields of type ports and the field id.orig_h,
you would run

	zar index -d /path/to/logs id.orig_h :port

Each pattern results a separate zdx bundle for each bzng file found.  The zdx
bundles foor a given bzng file are written to a sub-directory of the directory
containing that file, where the name of the sub-directory is a concatenation
of the bzng file name and the suffix ".zar".
`,
	New: New,
}

func init() {
	root.Zar.Add(Index)
}

type Command struct {
	*root.Command
	dir string
}

func New(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	c := &Command{Command: parent.(*root.Command)}
	f.StringVar(&c.dir, "d", "", "directory to descend")
	return c, nil
}

func (c *Command) Run(args []string) error {
	if len(args) == 0 {
		return errors.New("zar index: one or more indexing patterns must be specified")
	}
	if c.dir == "" {
		return errors.New("zar index: a directory must be specified with -d")
	}
	var rules []archive.Rule
	for _, pattern := range args {
		rule, err := archive.NewRule(pattern)
		if err != nil {
			return errors.New("zar index: " + err.Error())
		}
		rules = append(rules, rule)
	}
	return archive.IndexDirTree(c.dir, rules)
}
